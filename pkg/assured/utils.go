package assured

import "errors"

func convertExpectedCallToCall(ec *ExpectedCall, bodyIndex *int) (*Call, error) {
	call := Call{
		Path:       ec.Path,
		Method:     ec.Method,
		StatusCode: ec.StatusCode,
		Delay:      ec.Delay,
		Headers:    ec.Headers,
		Query:      ec.Query,
		Response:   ec.Response,
		Callbacks:  ec.Callbacks,
	}
	if bodyIndex != nil {
		if len(*ec.OrderedBodies) > 0 {
			call.Body = (*ec.OrderedBodies)[*bodyIndex]
		} else {
			return nil, errors.New("body requested but no ordered bodies found")
		}
	}

	return &call, nil
}
