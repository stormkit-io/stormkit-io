# Coding Conventions

## Go

### Package & File Layout

- Handler files: `handler_<resource>_<action>.go` (e.g. `handler_app_create.go`).
- Every handler file has a sibling test file: `handler_<resource>_<action>_test.go`.
- Core / utility files: `request_context.go`, `services.go`, `validators.go`.
- Routes are declared in `services.go`

### Handler Signature

```go
func handler<Name>(req *RequestContext) *shttp.Response
```

- Always takes `*RequestContext` (or `*RequestContextMCP` for MCP tool wrappers).
- Always returns `*shttp.Response` — never a `(value, error)` tuple.

### Request Parsing

**Path parameters** — use compact `if` with initializer:

```go
if id := utils.StringToID(req.Vars()["id"]); id == 0 {
    return shttp.NotFound()
}
```

**Query parameters** — use `Validators` helpers:

```go
v := &Validators{}
errs := []string{}

if from, err := v.ToInt(req.Query().Get("from"), "from"); err != nil {
    errs = append(errs, err.Error())
}

if repo, ok := v.NormalizeRepo(req.Query().Get("repo")); !ok {
    errs = append(errs, "repo must match 'provider/org/repo' format")
}

if len(errs) > 0 {
    return shttp.BadRequest(map[string]any{"errors": errs})
}
```

**JSON body**:

```go
type myResource struct {
    Name string  `json:"name"`
    Note *string `json:"note,omitempty"` // pointer for optional/PATCH fields
}

data := &myResource{}

if err := req.Post(data); err != nil {
    return shttp.Error(err)
}
```

Use pointer fields for optional `PUT`/`PATCH` fields so absence can be distinguished from zero value.

**Multipart form**:

```go
if req.MultipartForm == nil {
    return shttp.BadRequest(map[string]any{"errors": []string{"Expected multipart/form-data"}})
}

files := req.MultipartForm.File["files"]
```

### Responses

Use `shttp` helpers:

```go
shttp.OK()                     // 200, empty body
shttp.Error(err)               // 500, logs error
shttp.Error(err, "msg")        // 500, custom log message
shttp.BadRequest(data)         // 400
shttp.NotFound()               // 404
shttp.Forbidden()              // 403
shttp.NotAllowed()             // 401
```

Use a literal `*shttp.Response` when you need a specific status code:

```go
return &shttp.Response{
    Status: http.StatusCreated,
    Data: map[string]any{
        "id": resource.ID
    },
}
```

**Validation errors** should follow the existing response shapes in this package. For a single validation error, use an `"error"` string; when returning multiple messages, use an `"errors"` key with a string slice:

```go
// Single validation error (most common for param/query failures)
return shttp.BadRequest(map[string]any{
    "error": "Name is required",
})

// Multiple validation errors (domain/model validation)
return shttp.BadRequest(map[string][]string{
    "errors": {"Name is required", "Branch is invalid"},
})
```

**Duplicate / conflict**:

```go
if err := store.Insert(ctx, data); err != nil {
    if database.IsDuplicate(err) {
        return &shttp.Response{
            Status: http.StatusConflict,
            Data:   map[string][]string{"errors": {"Resource already exists"}},
        }
    }

    return shttp.Error(err)
}
```

### Route Registration (`services.go`)

```go
func Services(r *shttp.Router) *shttp.Service {
    s := r.NewService()

    s.NewEndpoint("/v1/app").
        Handler(shttp.MethodPost, "", WithAPIKey(handlerAppCreate, &Opts{MinimumScope: apikey.SCOPE_TEAM})).
        Handler(shttp.MethodGet,  "", WithAPIKey(handlerAppGet,    &Opts{MinimumScope: apikey.SCOPE_APP}))

    // Sub-path as the second argument to Handler
    s.NewEndpoint("/v1/app").
        Handler(shttp.MethodGet, "/config", WithAPIKey(handlerAppConf, &Opts{MinimumScope: apikey.SCOPE_APP}))

    // Path parameter (gorilla mux regex)
    s.NewEndpoint("/v1/deployments/{id:[0-9]+}").
        Handler(shttp.MethodGet, "", WithAPIKey(handlerDeploymentGet, &Opts{MinimumScope: apikey.SCOPE_ENV}))

    // Endpoint-level middleware (e.g. enterprise gate, upload limits)
    s.NewEndpoint("/v1/domains").
        Middleware(user.WithEE).
        Handler(shttp.MethodPut, "/cert", WithAPIKey(...))

    return s
}
```

- Prefer a single `NewEndpoint()` per base path; chain `.Handler()` for multiple HTTP methods.
- It is acceptable to register the same base path with multiple `NewEndpoint()` calls when endpoint-level middleware must differ (e.g. `/v1/domains` has a second registration with `Middleware(user.WithEE)` for the EE-only `/cert` sub-paths).
- Always wrap handlers with `WithAPIKey(handler, &Opts{MinimumScope: <scope>})`.

### Authentication & Scope

Scope constants (`apikey` package) represent the type of identifier a key carries. Each key has access only to the object it was created for:

| Constant            | Key must carry                                       |
| ------------------- | ---------------------------------------------------- |
| `apikey.SCOPE_USER` | UserID — access to everything the user has access to |
| `apikey.SCOPE_TEAM` | TeamID (or user who is a team member)                |
| `apikey.SCOPE_APP`  | AppID (or user/team with app access)                 |
| `apikey.SCOPE_ENV`  | EnvID — access to that environment only              |

Note: these scopes represent distinct key *types*, not a permission hierarchy. `WithAPIKey` does not compare scope values numerically; it checks that the key carries the required identifier for the requested resource.

`WithAPIKey` populates `req.Token`, `req.App`, `req.Env`, and `req.TeamID` on the `RequestContext`.

### Validation

Call domain-package validators before persisting:

```go
if errs := app.Validate(myApp); len(errs) > 0 {
    return shttp.BadRequest(map[string][]string{"errors": errs})
}

if errs := buildconf.Validate(env); len(errs) > 0 {
    return shttp.BadRequest(map[string][]string{"errors": errs})
}
```

### Pagination

Offset-based; use `limit+1` to detect a next page:

```go
const myListLimit = 20

results, err := store.List(ctx, from, myListLimit+1)
hasNextPage := len(results) > myListLimit

if hasNextPage {
    results = results[:myListLimit]
}

return &shttp.Response{
    Status: http.StatusOK,
    Data: map[string]any{
        "results":     results,
        "hasNextPage": hasNextPage,
    },
}
```

### Audit Logging (Enterprise)

Log all mutations when `req.License().IsEnterprise()`:

```go
if req.License().IsEnterprise() {
    err := audit.FromRequestContext(req).
        WithAction(audit.CreateAction, audit.TypeApp).
        WithDiff(&audit.Diff{
            Before: audit.DiffFields{"name": old.Name},
            After:  audit.DiffFields{"name": new.Name},
        }).
        Insert()
    if err != nil {
        return shttp.Error(err)
    }
}
```

TeamID, AppID, EnvID, UserID are all used in auditing. If one of these is populated
within the handler, include them using the helpers:

```go
teamID := populatedInsidedTheHandler()

err := audit.FromRequestContext(req).
        WithAction(audit.CreateAction, audit.TypeApp).
        WithDiff(&audit.Diff{
            Before: audit.DiffFields{"name": old.Name},
            After:  audit.DiffFields{"name": new.Name},
        }).
        WithTeamID(teamID)
        Insert()
```

Otherwise, the `FromRequestContext` method gathers all the context from the request.

Capture old state before mutations so `Before`/`After` diffs are accurate.

### Testing Setup

- Package: `publicapiv1_test` (external test package).
- Embed `*factory.Factory` in the suite for test-data helpers.
- Wrap the DB in a transaction per test via `databasetest.InitTx` / `conn.CloseTx`.

```go
type HandlerAppCreateSuite struct {
    suite.Suite
    *factory.Factory
    conn databasetest.TestDB
}

func (s *HandlerAppCreateSuite) BeforeTest(suiteName, _ string) {
    s.conn = databasetest.InitTx(suiteName)
    s.Factory = factory.New(s.conn)
}

func (s *HandlerAppCreateSuite) AfterTest(_, _ string) {
    s.conn.CloseTx()
}

func (s *HandlerAppCreateSuite) handler() http.Handler {
    return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}
```

Use `shttptest.RequestWithHeaders` to fire requests; assert with `response.Code` and `response.Map()`.

### Testing Conventions

**Test method naming** — `Test_<Scenario>` or `Test_<Scenario>_<Detail>`:

```go
func (s *MySuite) Test_Success() { ... }
func (s *MySuite) Test_Success_UserScopedKey() { ... }
func (s *MySuite) Test_Forbidden_NotMember() { ... }
func (s *MySuite) Test_NotFound_UnknownID() { ... }
```

**Test doc comments** — non-obvious test methods get a one-line comment:

```go
// Test_NotFound_WrongEnv verifies that a deployment belonging to a different env is not accessible.
func (s *MySuite) Test_NotFound_WrongEnv() { ... }
```

**Suite helper methods** — extract repeated request setup into named helpers on the suite:

```go
// post sends a POST to /v1/mcp with the given body and Authorization header.
func (s *MySuite) post(keyValue string, body any) shttptest.Response {
    return shttptest.RequestWithHeaders(s.handler(), shttp.MethodPost, "/v1/mcp", body,
        map[string]string{"Authorization": keyValue})
}

// userKey creates a SCOPE_USER API key owned by usr.
func (s *MySuite) userKey(usr *factory.MockUser) *factory.MockAPIKey {
    return s.MockAPIKey(nil, nil, map[string]any{
        "UserID": usr.ID,
        "Scope":  apikey.SCOPE_USER,
    })
}
```

**`Require` vs `assert`** — use `s.Require().NoError` for setup steps that must not fail; use plain `s.NoError` / `s.Equal` for assertions:

```go
conf, err := json.Marshal(snapshot)
s.Require().NoError(err) // setup — abort test on failure

s.Equal(http.StatusOK, response.Code) // assertion
s.Equal("running", body["status"])
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type: short description in lowercase

- Optional bullet for each notable change
- Another bullet
```

Types used in this project:

| Type    | When to use                              |
| ------- | ---------------------------------------- |
| `feat`  | New user-facing feature or endpoint      |
| `fix`   | Bug fix                                  |
| `chore` | Maintenance, refactoring, tests, tooling |
| `docs`  | Documentation                            |

### Go Doc Comments

Exported functions and non-obvious helpers get a doc comment starting with the function name:

```go
// DeploymentStatus returns a lightweight deployment record containing only the fields
// needed to compute Status(). Returns nil (no error) when no matching deployment exists.
func (s *Store) DeploymentStatus(...) (*Deployment, error) { ... }
```

### Code Style

Always leave a blank line between logical blocks (variable declarations, `if` statements, `for` loops, `return` statements, etc.) to improve readability:

```go
// Good
envID := utils.StringToID(req.Vars()["id"])

if envID == 0 {
    return shttp.NotFound()
}

env, err := store.EnvironmentByID(ctx, envID)

if err != nil {
    return shttp.Error(err)
}

return shttp.OK()

// Avoid
envID := utils.StringToID(req.Vars()["id"])
if envID == 0 {
    return shttp.NotFound()
}
env, err := store.EnvironmentByID(ctx, envID)
if err != nil {
    return shttp.Error(err)
}
return shttp.OK()
```

### Common Internal Imports

```go
// Domain / API
"github.com/stormkit-io/stormkit-io/src/ce/api/app"
"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
"github.com/stormkit-io/stormkit-io/src/ee/api/team"

// Libraries
"github.com/stormkit-io/stormkit-io/src/lib/shttp"
"github.com/stormkit-io/stormkit-io/src/lib/database"
"github.com/stormkit-io/stormkit-io/src/lib/utils"
"github.com/stormkit-io/stormkit-io/src/lib/types"
"github.com/stormkit-io/stormkit-io/src/lib/slog"
```
