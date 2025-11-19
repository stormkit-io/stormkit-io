package runner

import (
	"context"
	"crypto/sha1"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

//go:embed assets/stormkit-api.mjs
var APIWrapper string

const FindDependencyRegexp = `(` + // opening bracket
	// all possible import and require statements:
	`import\s+([:\w{}\s\*,'"]+\s+from\s+)?|` +
	`require\(|` +
	`import\(` +
	// closing bracket
	`)` +
	// quotest start
	`['"]` +
	// what we're looking for: the module name
	`([\w@\/\-]+)` +
	// quotes end
	`['"]`

const StormkitTmpFolder = "stormkit-tmp"
const StormkitServerFolder = ".stormkit/server"
const StormkitPublicFolder = ".stormkit/public"
const StormkitAPIFolder = ".stormkit/api"

type BundlerInterface interface {
	Zip(*Artifacts) error
	Bundle(context.Context) (*Artifacts, error)
	ParseRedirects(*Artifacts) error
	ParseHeaders(*Artifacts) error
}

var DefaultBundler BundlerInterface

type Bundler struct {
	workDir       string   // Absolute path
	repoDir       string   // Absolute path
	distDir       string   // Absolute path where the zip files will be uploaded (not the same with Build.DistDir)
	clientDirs    []string // The directories to look for client-side files
	serverDirs    []string // The directories to look for server-side files
	apiDirs       []string // The directories to look for api files
	serverCmd     string   // The command to spin up the Node.js server
	headersFile   string   // Relative path to the headers file (from working dir)
	redirectsFile string   // Relative path to the redirects file (from working dir)
	apiFolder     string   // Relative path to the api dir (from working dir)
	packageJson   *PackageJson
	reporter      *ReporterModel
}

// NewArtifacts creates a new artifacts object. This is mainly used for tests.
func NewArtifacts(workDir string) *Artifacts {
	return &Artifacts{
		workDir: workDir,
	}
}

type Artifacts struct {
	workDir        string
	distDir        string
	clientZip      string // absolute path to client zip
	serverZip      string // absolute path to server zip
	apiZip         string // absolute path to api zip
	isAPIAutoBuilt bool

	// List of redirects.
	Redirects []deploy.Redirect

	// List of client directories that will be uploaded to the CDN.
	ClientDirs []string

	// List of server directories that will be uploaded to the renderer function.
	ServerDirs []string

	// List of api directories that will be uploaded to the api function.
	ApiDirs []string

	// Entry file and exported function for the serverless entry file in
	// file:handler name format. For instance, index.mjs:handler.
	FunctionHandler string

	// Entry file and exported function for the api entry file in file:handler
	// name format. For instance, index.mjs:handler.
	ApiHandler string

	// Key value object for headers. This will used to map CDN files.
	// Example:
	// "/index.html": map[string]string{ "x-my-header": "value" }
	Headers []deploy.CustomHeader
}

// APIFiles returns a list of api files to be included in the manifest.
// Spec files, private files (starting with an underscore) and directories are ignored.
func (a *Artifacts) APIFiles() []deploy.APIFile {
	files := []deploy.APIFile{}

	if a == nil || a.ApiDirs == nil {
		return files
	}

	for _, dir := range a.ApiDirs {
		fullPath := filepath.Join(a.workDir, dir)

		if !file.Exists(fullPath) {
			continue
		}

		_ = filepath.WalkDir(fullPath, func(pathToFile string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			fileName := strings.Replace(pathToFile, fullPath, "", 1)

			if pathToFile == dir || info.IsDir() || fileName == "" {
				return nil
			}

			if strings.HasPrefix(fileName, "/_") ||
				!strings.HasSuffix(fileName, "js") ||
				strings.Contains(fileName, ".spec.") ||
				strings.HasPrefix(fileName, "/stormkit-api.") { // .js or .mjs file
				return nil
			}

			files = append(files, deploy.APIFile{
				FileName: fileName,
			})

			return nil
		})
	}

	return files
}

// CDNFiles returns a list of files with the etag header.
// This is used to include in the manifest.
func (a *Artifacts) CDNFiles() []deploy.CDNFile {
	files := []deploy.CDNFile{}
	cache := map[string]bool{}

	if a == nil || a.ClientDirs == nil {
		return files
	}

	for _, dir := range a.ClientDirs {
		fullPath := filepath.Join(a.workDir, dir)

		if !file.Exists(fullPath) {
			continue
		}

		_ = filepath.WalkDir(fullPath, func(pathToFile string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			fileName := strings.Replace(pathToFile, fullPath, "", 1)

			if pathToFile == dir || info.IsDir() || fileName == "" {
				return nil
			}

			if a.isAPIAutoBuilt && strings.HasPrefix(fileName, "/api") {
				return nil
			}

			if cache[fileName] {
				return nil
			}

			headers := deploy.ApplyHeaders(
				fileName,
				map[string]string{"etag": etag(pathToFile, false)},
				a.Headers,
			)

			files = append(files, deploy.CDNFile{
				Name:    fileName,
				Headers: headers,
			})

			// This will prevent adding the same file
			cache[fileName] = true

			return nil
		})
	}

	return files
}

func findDistDir(opts RunnerOpts) string {
	if opts.Build.DistFolder != "" {
		return opts.Build.DistFolder
	}

	candidates := []string{"dist", "build", "output", "out"}

	for _, dir := range candidates {
		if file.Exists(filepath.Join(opts.WorkDir, dir)) {
			return dir
		}
	}

	return ""
}

// distDirs determines the client, server and api directories to deploy.
//
// Case 1: `.stormkit` folder exists
//
// - client: .stormkit/public
// - server: .stormkit/server
// - api: .stormkit/api
//
// Case 2: custom dist folder exists
//
// - client: <dist-folder>/public | static | client
// - server: <dist-folder>/server
// - api: <dist-folder>/api
func distDirs(opts RunnerOpts) ([]string, []string, []string) {
	clientDirs := []string{}
	serverDirs := []string{}
	apiDirs := []string{}

	// If the repository exposes a public folder, include it by default
	if file.Exists(filepath.Join(opts.WorkDir, "public")) {
		clientDirs = append(clientDirs, "public")
	}

	// The `.stormkit/api` folder has the highest precedence, because we
	// build this folder automatically
	if file.Exists(filepath.Join(opts.WorkDir, ".stormkit", "api")) {
		apiDirs = append(apiDirs, StormkitAPIFolder)
	}

	hasStormkitPublicSubfolder := file.Exists(filepath.Join(opts.WorkDir, ".stormkit", "public"))
	hasStormkitServerSubfolder := file.Exists(filepath.Join(opts.WorkDir, ".stormkit", "server"))

	if hasStormkitPublicSubfolder {
		clientDirs = append(clientDirs, StormkitPublicFolder)
	}

	if hasStormkitServerSubfolder {
		serverDirs = append(serverDirs, StormkitServerFolder)
	}

	// If we have a `.stormkit/public` or `.stormkit/server` subfolder, return early
	if hasStormkitPublicSubfolder || hasStormkitServerSubfolder {
		return clientDirs, serverDirs, apiDirs
	}

	// Otherwise look for deploying the dist folder
	// Either we have a dist/server + dist/public structure or just a dist folder
	distDir := findDistDir(opts)

	// We're deploying a server application in this case
	if opts.Build.ServerCmd != "" || opts.Build.ServerFolder != "" {
		return clientDirs, []string{utils.GetString(distDir, opts.Build.ServerFolder)}, apiDirs
	}

	if distDir == "" {
		return clientDirs, serverDirs, apiDirs
	}

	publicSubfolders := []string{"public", "static", "client", "browser"}
	changed := false

	for _, subfolder := range publicSubfolders {
		if file.Exists(filepath.Join(opts.WorkDir, distDir, subfolder)) {
			clientDirs = append(clientDirs, filepath.Join(distDir, subfolder))
			changed = true
			break
		}
	}

	if file.Exists(filepath.Join(opts.WorkDir, distDir, "server")) {
		serverDirs = append(serverDirs, filepath.Join(distDir, "server"))
		changed = true
	}

	// In this case we deploy the dist folder completely as client-side files
	if !changed {
		clientDirs = append(clientDirs, distDir)
	}

	return clientDirs, serverDirs, apiDirs
}

func NewBundler(opts RunnerOpts) BundlerInterface {
	distDir := filepath.Join(opts.RootDir, "dist")

	// Just in case it does not exist
	_ = os.MkdirAll(distDir, 0776)

	clientDirs, serverDirs, apiDirs := distDirs(opts)

	return Bundler{
		repoDir:       opts.Repo.Dir,
		workDir:       opts.WorkDir,
		distDir:       distDir,
		clientDirs:    clientDirs,
		serverDirs:    serverDirs,
		apiDirs:       apiDirs,
		headersFile:   opts.Build.HeadersFile,
		redirectsFile: opts.Build.RedirectsFile,
		serverCmd:     opts.Build.ServerCmd,
		apiFolder:     opts.Build.APIFolder,
		packageJson:   opts.Repo.PackageJson,
		reporter:      opts.Reporter,
	}
}

// Bundle takes the deployment folder and prepares the zip files.
// By default, this function will look at the .stormkit folder in the
// working directory (which is specified through the SK_CWD env variable).
//
// The .stormkit folder has the following structure:
// - public
// - server
// - api
//
// If instead of the .stormkit folder, another dist folder respecting the same
// output folders is given, that will also work properly.
//
// If the custom dist folder does not have a public|server|api subfolder, the whole
// content of the dist folder will be uploaded to S3 bucket.
//
// Otherwise, this function will lookup for the following folders:
//
// - out
// - output
// - dist
// - build
func (b Bundler) Bundle(ctx context.Context) (*Artifacts, error) {
	artifacts := &Artifacts{
		workDir: b.workDir,
		distDir: b.distDir,
	}

	var err error

	artifacts.ServerDirs, artifacts.FunctionHandler, err = b.bundleServerSide()

	if err != nil {
		return nil, err
	}

	artifacts.ApiDirs, artifacts.ApiHandler, err = b.bundleApiFolder(ctx)

	if err != nil {
		return nil, err
	}

	artifacts.ClientDirs, err = b.bundleClientSide()

	if err != nil {
		return nil, err
	}

	// If nothing is found, deploy the whole folder.
	if len(artifacts.ServerDirs) == 0 && len(artifacts.ClientDirs) == 0 && len(artifacts.ApiDirs) == 0 {
		artifacts.ClientDirs = []string{"."}
	}

	return artifacts, nil
}

// findServerDependencies will look at the commands and find the dependencies required for
// running the server command.
func (b Bundler) findServerDependencies(commands []utils.Command) []string {
	deps := []string{}

	for _, cmd := range commands {
		if cmd.IsPackageManager && b.packageJson != nil {
			if resolvedCmd := b.packageJson.Scripts[cmd.ScriptName]; resolvedCmd != "" {
				deps = append(deps, b.findServerDependencies(utils.ParseCommands(resolvedCmd))...)
			}
		} else {
			deps = append(deps, cmd.CommandName)
		}
	}

	return deps
}

// bundleServerSideStormkitSubfolder bundles the server side code when the built files
// are located inside `.stormkit/server` folder. This is mainly used for serverless deployments.
func (b Bundler) bundleServerSideStormkitSubfolder() ([]string, string, error) {
	pathToDist := filepath.Join(b.workDir, b.serverDirs[0])
	serverEntry, functionHandler := autoDetectServerFile(pathToDist)

	if serverEntry == "" {
		return nil, "", fmt.Errorf("cannot auto detect serverless entry file: expecting a file name called (index|server).{js,mjs,cjs}")
	}

	dependencies, err := b.bundleDependencies(pathToDist)

	if err != nil {
		return nil, "", err
	}

	// Move required node_modules to .stormkit/server/node_modules
	// This is because in this case we deploy only `.stormkit/server` folder
	for _, dep := range dependencies {
		err := file.Rsync(file.RsyncArgs{
			Context:     context.Background(),
			Source:      dep,
			Destination: StormkitServerFolder,
			WorkDir:     b.workDir,
		})

		if err != nil {
			slog.Errorf("error while copying dependency %s: %s", dep, err.Error())
		}
	}

	return []string{StormkitServerFolder}, fmt.Sprintf("%s:%s", serverEntry, functionHandler), nil
}

// bundleServerSide returns the necessary information to bundle the server side code.
func (b Bundler) bundleServerSide() ([]string, string, error) {
	if b.serverCmd == "" && len(b.serverDirs) == 0 {
		return nil, "", nil
	}

	functionHandler := ".:server"

	// Handle bundling the whole folder case (go, python, ruby, etc...)
	if len(b.serverDirs) == 1 && b.serverDirs[0] == "" {
		return []string{"."}, functionHandler, nil
	}

	// Handle .stormkit/server subfolder case (serverless)
	if len(b.serverDirs) == 1 && b.serverDirs[0] == StormkitServerFolder {
		return b.bundleServerSideStormkitSubfolder()
	}

	// Handle case with server command
	serverDirs := []string{}
	commandDeps := b.findServerDependencies(utils.ParseCommands(b.serverCmd))
	additionalDeps := []string{}

	for _, commandDep := range commandDeps {
		target, err := os.Readlink(filepath.Join(b.workDir, "node_modules", ".bin", commandDep))

		if os.IsNotExist(err) || target == "" {
			continue
		}

		var relativePath string

		if strings.HasPrefix(target, "../") {
			relativePath = target[len("../"):]
		} else {
			relativePath = strings.Replace(target, b.workDir, "", 1)
		}

		// Split the remaining path into segments
		segments := strings.Split(relativePath, "/")
		packageName := ""

		// Determine the package name
		if len(segments) > 0 {
			if strings.HasPrefix(segments[0], "@") && len(segments) > 1 {
				// Scoped package (e.g., @remix-run/serve)
				packageName = filepath.Join(segments[0], segments[1])
			} else {
				// Regular package (e.g., express)
				packageName = segments[0]

			}
		}

		additionalDeps = append(additionalDeps, packageName)
	}

	for _, dir := range b.serverDirs {
		absolutePath := filepath.Join(b.workDir, dir)

		if dir == "" || !file.Exists(absolutePath) {
			continue
		}

		deps, err := b.bundleDependencies(absolutePath, additionalDeps...)

		if err != nil {
			slog.Errorf("error while bundling dependencies: %s", err.Error())
			continue
		}

		serverDirs = append(serverDirs, deps...)
		serverDirs = append(serverDirs, trim(dir))
	}

	return serverDirs, functionHandler, nil
}

// bundleApiFolder will look at <relative-dist>/api folder
// and return the necessary information when the folder is found. This function
// also bundles dependencies by looking at the top-level api dir.
func (b Bundler) bundleApiFolder(ctx context.Context) ([]string, string, error) {
	for _, dir := range b.apiDirs {
		absDir := filepath.Join(b.workDir, dir)

		if dir == "" {
			continue
		}

		if file.Exists(absDir) {
			err := os.WriteFile(filepath.Join(absDir, "stormkit-api.mjs"), []byte(APIWrapper), 0664)

			if err != nil {
				return nil, "", err
			}

			if err := b.BundleDependencies(ctx, absDir); err != nil {
				return nil, "", err
			}

			return []string{dir}, "stormkit-api.mjs:handler", nil
		}
	}

	return nil, "", nil
}

func (b Bundler) bundleClientSide() ([]string, error) {
	retVal := []string{}

	for _, dir := range b.clientDirs {
		if file.Exists(filepath.Join(b.workDir, dir)) {
			retVal = append(retVal, dir)
		}
	}

	return retVal, nil
}

// findDependencies finds the dependencies that are used in the server folder.
func (b Bundler) findDependencies(sourceFolder, pattern string) ([]string, error) {
	matches := make(map[string][]string)

	// Compile the provided regex pattern
	re, err := regexp.Compile(pattern)

	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Walk through the source folder
	err = filepath.Walk(sourceFolder, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file has a .js, .mjs, or .cjs extension
		if !info.IsDir() && (filepath.Ext(path) == ".js" || filepath.Ext(path) == ".mjs" || filepath.Ext(path) == ".cjs") {

			// Read the file content
			content, err := os.ReadFile(path)

			if err != nil {
				return err
			}

			fileMatches := re.FindAllStringSubmatch(RemoveJSComments(string(content)), -1)

			for _, match := range fileMatches {
				if lm := len(match); lm > 0 {
					matches[path] = append(matches[path], match[lm-1])
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	deps := []string{}
	keys := []string{}

	for k := range b.packageJson.Dependencies {
		keys = append(keys, k)
	}

	for k := range b.packageJson.DevDependencies {
		keys = append(keys, k)
	}

	for _, match := range matches {
		for _, m := range match {
			if utils.InSliceString(keys, m) && !utils.InSliceString(deps, m) {
				deps = append(deps, m)
			}
		}
	}

	return deps, nil
}

// bundleDependencies returns a list of node_modules that are going to be bundled with the server.
func (b Bundler) bundleDependencies(entryFile string, includedDeps ...string) ([]string, error) {
	if b.packageJson == nil {
		return nil, nil
	}

	foundDeps, err := b.findDependencies(entryFile, FindDependencyRegexp)

	if err != nil {
		return nil, err
	}

	foundDeps = append(foundDeps, includedDeps...)

	bundledDependencies := []string{}
	bundledDependencies = append(bundledDependencies, b.packageJson.BundleDependencies...)
	bundledDependencies = append(bundledDependencies, foundDeps...)

	dependenciesMap := map[string]bool{}

	for _, dep := range bundledDependencies {
		dependenciesMap[fmt.Sprintf("node_modules/%s", dep)] = true
	}

	dt := NewDepedencyTree(bundledDependencies, filepath.Join(b.workDir, "node_modules"))
	dt.Walk()

	resolved := dt.ResolvedDepedencies()

	for _, dep := range resolved {
		dependenciesMap[fmt.Sprintf("node_modules/%s", dep.Name)] = true
	}

	dependencies := []string{
		"package.json",
		"node_modules/.bin",
	}

	for k := range dependenciesMap {
		dependencies = append(dependencies, k)
	}

	return dependencies, nil
}

// bundleDependencies will traverse the dist folder, find all imported/required dependencies
// and include them in the bundle.
func (b Bundler) BundleDependencies(ctx context.Context, destination string, includedDeps ...string) error {
	if b.packageJson == nil {
		return nil
	}

	foundDeps, err := b.findDependencies(destination, FindDependencyRegexp)

	if err != nil {
		return err
	}

	foundDeps = append(foundDeps, includedDeps...)

	bundledDependencies := []string{}
	bundledDependencies = append(bundledDependencies, b.packageJson.BundleDependencies...)
	bundledDependencies = append(bundledDependencies, foundDeps...)

	if len(bundledDependencies) == 0 {
		return nil
	}

	b.reporter.AddStep("bundling server packages")

	pathToNodeModules := filepath.Join(b.workDir, "node_modules")
	dt := NewDepedencyTree(bundledDependencies, pathToNodeModules)
	dt.Walk()

	resolved := dt.ResolvedDepedencies()
	nodeModules := filepath.Join(destination, "node_modules")

	if !file.Exists(nodeModules) {
		if err := os.MkdirAll(nodeModules, 0776); err != nil {
			return err
		}
	}

	slices.Sort(foundDeps)

	for _, dep := range resolved {
		parentFolder := nodeModules

		// If the name is something like `@swc/helpers` we need to prepare the folder
		// beforehand and upload the files inside that folder.
		if strings.Contains(dep.Name, "/") {
			if err := os.MkdirAll(filepath.Join(nodeModules, dep.Name), 0776); err != nil {
				slog.Errorf("error while making folder: %s", err.Error())
				continue
			}

			parentFolder = filepath.Join(nodeModules, filepath.Dir(dep.Name))
		}

		cmd := exec.CommandContext(ctx, "cp", "-R", dep.FullPath, parentFolder)
		cmd.Dir = b.workDir
		cmd.Stdout = b.reporter.File()
		cmd.Stderr = nil // Otherwise we include lines like: No such file or directory
		cmd.Env = []string{
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		}

		if err := cmd.Run(); err != nil {
			slog.Infof("[warning]: bundling server package %s ignored because %s", dep.Name, err.Error())
		}
	}

	for _, dep := range foundDeps {
		b.reporter.AddLine(dep)
	}

	return nil
}

// Zip all artifacts
func (b Bundler) Zip(artifacts *Artifacts) error {
	zip := func(zipName string, dirs []string, includeParent bool) error {
		workingDir := b.workDir

		return file.ZipV2(file.ZipArgs{
			Source:        dirs,
			ZipName:       zipName,
			WorkingDir:    workingDir,
			IncludeParent: includeParent,
		})
	}

	clientZip := filepath.Join(b.distDir, "sk-client.zip")
	serverZip := filepath.Join(b.distDir, "sk-server.zip")
	apiZip := filepath.Join(b.distDir, "sk-api.zip")

	if err := zip(clientZip, artifacts.ClientDirs, false); err != nil {
		return err
	}

	includeParentFolder := true

	if len(artifacts.ServerDirs) != 0 && artifacts.ServerDirs[0] == StormkitServerFolder {
		includeParentFolder = false
	}

	if err := zip(serverZip, artifacts.ServerDirs, includeParentFolder); err != nil {
		return err
	}

	if err := zip(apiZip, artifacts.ApiDirs, false); err != nil {
		return err
	}

	if len(artifacts.ClientDirs) > 0 {
		artifacts.clientZip = clientZip
	}

	if len(artifacts.ServerDirs) > 0 {
		artifacts.serverZip = serverZip
	}

	if len(artifacts.ApiDirs) > 0 {
		artifacts.apiZip = apiZip
	}

	return nil
}

// ParseHeaders will parse the headers file and update
// artifacts objects with the headers. This requires the
// `headersFile` property to be set on the deployment object.
func (b Bundler) ParseHeaders(artifacts *Artifacts) error {
	if b.headersFile == "" {
		return nil
	}

	pathToFile := filepath.Join(b.workDir, b.headersFile)
	headers, err := deploy.ParseHeadersFile(pathToFile)

	if err != nil {
		b.reporter.AddStep("parsing headers file failed")
		b.reporter.AddLine(fmt.Sprintf("File not found: %s", pathToFile))
		return err
	}

	// Do not block the deployment but notify the user about the failed step
	if pathToFile != "" && len(headers) == 0 {
		slog.Infof("warning: headers file %s is specified but no headers were found", pathToFile)
		return nil
	}

	artifacts.Headers = headers

	return nil
}

// ParseRedirects will parse the redirects.json file and update
// artifacts objects with the redirects. If speficied, this function
// will also look at the redirectsFile.
//
// Both the working directory (which is set through SK_CWD env variable)
// and repository root are checked for the redirects.json. The precedence
// goes to the working directory.
//
// This function will also Netlify style _redirects. The same logic about
// directory order applies to Netlify style _redirects as well.
func (b Bundler) ParseRedirects(artifacts *Artifacts) error {
	files := []string{}

	if b.redirectsFile != "" {
		files = append(files, filepath.Join(b.workDir, b.redirectsFile))
	} else {
		files = append(files,
			filepath.Join(b.workDir, "redirects.json"),
			filepath.Join(b.repoDir, "redirects.json"),
			filepath.Join(b.workDir, "_redirects"),
			filepath.Join(b.repoDir, "_redirects"),
		)
	}

	for _, f := range files {
		if file.Exists(f) {
			data, err := os.ReadFile(f)

			if err != nil {
				return err
			}

			if strings.HasSuffix(f, ".json") {
				artifacts.Redirects = []redirects.Redirect{}

				if err := json.Unmarshal(data, &artifacts.Redirects); err != nil {
					return err
				}

				continue
			}

			if err := b.parseNetlifyRedirects(artifacts, f); err != nil {
				return err
			}

			return nil
		}
	}

	return nil
}

func (b Bundler) parseNetlifyRedirects(artifacts *Artifacts, file string) error {
	doc, err := os.ReadFile(file)

	if err != nil {
		return err
	}

	lines := strings.Split(string(doc), "\n")
	artifacts.Redirects = []deploy.Redirect{}

	for _, line := range lines {
		pieces := strings.Fields(line)

		// Invalid statement, ignore it.
		if len(pieces) < 2 {
			continue
		}

		redirect := deploy.Redirect{
			From: pieces[0],
			To:   strings.Replace(pieces[1], ":splat", "$1", 1),
		}

		if len(pieces) > 2 {
			redirect.Status, _ = strconv.Atoi(strings.ReplaceAll(pieces[2], "!", ""))
		}

		// Special case, make sure it's not a hard redirect.
		if strings.Contains(redirect.From, "*") && strings.HasSuffix(redirect.To, ".html") {
			redirect.Assets = false
			redirect.Status = 0
		} else if redirect.Status > 0 && string(strconv.Itoa(redirect.Status)[0]) != "3" {
			redirect.Status = 0
		} else if redirect.Status == 0 {
			redirect.Status = http.StatusMovedPermanently
		}

		artifacts.Redirects = append(artifacts.Redirects, redirect)
	}

	return nil
}

func etag(filePath string, weak bool) string {
	body, err := os.ReadFile(filePath)

	if err != nil {
		return ""
	}

	hash := sha1.Sum(body)
	etag := fmt.Sprintf("\"%d-%x\"", int(len(hash)), hash)

	if weak {
		etag = "W/" + etag
	}

	return etag
}

// trim removes the initial ./ part from the string.
func trim(s string) string {
	path := strings.TrimSpace(s)

	// Remove initial ./
	path = strings.TrimPrefix(path, "./")

	// Remove standalone .
	if path == "." {
		path = ""
	}

	// Remove initial /
	return strings.TrimPrefix(path, "/")
}

// RemoveJSComments uses regex to remove both single-line and multi-line comments
// from the given `code` string.
func RemoveJSComments(code string) string {
	// Remove single-line comments
	singleLineRegex := regexp.MustCompile(`//.*`)
	code = singleLineRegex.ReplaceAllString(code, "")

	// Remove multi-line comments
	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	code = multiLineRegex.ReplaceAllString(code, "")

	return strings.TrimSpace(code)
}

// autoDetectServerFile returns the server entry file name and the handler name.
func autoDetectServerFile(pathToServerlessFolder string) (string, string) {
	lookupFiles := []string{
		"index",
		"server",
	}

	extensions := []string{
		".js",
		".mjs",
		".cjs",
		".ts",
		".mts",
		".cts",
	}

	for _, lookupFile := range lookupFiles {
		for _, ext := range extensions {
			fileName := lookupFile + ext
			filePath := filepath.Join(pathToServerlessFolder, fileName)

			if file.Exists(filePath) {
				return fileName, "handler"
			}
		}
	}

	return "", ""
}
