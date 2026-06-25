package publicapiv1

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/adhocore/gronx"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type functionTriggerRequest struct {
	ID      types.ID `json:"id,string"`
	Cron    string   `json:"cron"`
	Status  bool     `json:"status"`
	Options struct {
		Headers shttp.Headers `json:"headers"`
		Method  string        `json:"method"`
		Payload string        `json:"payload"`
		URL     string        `json:"url"`
	} `json:"options"`
}

func validateFunctionTrigger(data *functionTriggerRequest) map[string]string {
	errors := map[string]string{}

	if !gronx.New().IsValid(data.Cron) {
		errors["cron"] = "Invalid cron format"
	}

	parsedURL, err := url.Parse(data.Options.URL)

	if err != nil ||
		parsedURL.Host == "" ||
		parsedURL.Scheme == "" ||
		!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		errors["url"] = "Invalid URL"
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func (r *functionTriggerRequest) toRecord(envID types.ID) *functiontrigger.FunctionTrigger {
	return &functiontrigger.FunctionTrigger{
		ID:     r.ID,
		Cron:   r.Cron,
		EnvID:  envID,
		Status: r.Status,
		Options: functiontrigger.Options{
			Method:  r.Options.Method,
			Headers: r.Options.Headers,
			URL:     r.Options.URL,
			Payload: []byte(r.Options.Payload),
		},
	}
}

func handlerFunctionTriggerCreate(req *RequestContext) *shttp.Response {
	tf := &functionTriggerRequest{}

	if err := req.Post(tf); err != nil {
		return shttp.Error(err)
	}

	if errs := validateFunctionTrigger(tf); errs != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   errs,
		}
	}

	record := tf.toRecord(req.Env.ID)
	record.ID = 0

	if err := functiontrigger.NewStore().Insert(req.Context(), record); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"trigger": record.ToMap(),
		},
	}
}
