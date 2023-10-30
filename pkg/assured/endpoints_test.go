package assured

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

var call1 = (*Call)(unsafe.Pointer(testCall1()))
var call2 = (*Call)(unsafe.Pointer(testCall2()))
var call3 = (*Call)(unsafe.Pointer(testCall3()))

func convertExpectedCallsToCalls(expectedCallStore *ExpectedCallStore) *CallStore {
	callsMade := make(map[string][]*Call, len(fullAssuredCalls.data))
	for key, expectedCalls := range fullAssuredCalls.data {
		calls := make([]*Call, len(expectedCalls))
		for _, call := range expectedCalls {
			calls = append(calls, (*Call)(unsafe.Pointer(&call)))
		}
		callsMade[key] = calls
	}

	callStore := NewCallStore()
	callStore.data = callsMade

	return callStore
}

func TestNewAssuredEndpoints(t *testing.T) {
	expected := &AssuredEndpoints{
		httpClient:     http.DefaultClient,
		assuredCalls:   NewExpectedCallStore(),
		madeCalls:      NewCallStore(),
		trackMadeCalls: true,
	}
	actual := NewAssuredEndpoints(DefaultOptions)

	require.Equal(t, expected.assuredCalls, actual.assuredCalls)
	require.Equal(t, expected.madeCalls, actual.madeCalls)
}

func TestWrappedEndpointSuccess(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)
	testEndpoint := func(ctx context.Context, call *Call) (interface{}, error) {
		return call, nil
	}

	actual := endpoints.WrappedEndpoint(testEndpoint)
	c, err := actual(context.TODO(), testCall1())

	require.NoError(t, err)
	require.Equal(t, testCall1(), c)
}

func TestWrappedEndpointFailure(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)
	testEndpoint := func(ctx context.Context, call *Call) (interface{}, error) {
		return call, nil
	}

	actual := endpoints.WrappedEndpoint(testEndpoint)
	c, err := actual(context.TODO(), false)

	require.Nil(t, c)
	require.Error(t, err)
	require.Equal(t, err.Error(), "unable to convert request to assured Call")
}

func TestGivenEndpointSuccess(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)

	c, err := endpoints.GivenEndpoint(context.TODO(), testCall1())

	require.NoError(t, err)
	require.Equal(t, testCall1(), c)

	c, err = endpoints.GivenEndpoint(context.TODO(), testCall2())

	require.NoError(t, err)
	require.Equal(t, testCall2(), c)

	c, err = endpoints.GivenEndpoint(context.TODO(), testCall3())

	require.NoError(t, err)
	require.Equal(t, testCall3(), c)

	require.Equal(t, fullAssuredCalls, endpoints.assuredCalls)
}

func TestGivenCallbackEndpointSuccess(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)

	callback1 := testCall1()
	callback1.Headers[AssuredCallbackKey] = "call-key"
	c, err := endpoints.GivenEndpoint(context.TODO(), callback1)

	require.NoError(t, err)
	require.Equal(t, callback1, c)

	callback2 := testCall2()
	callback2.Headers[AssuredCallbackKey] = "call-key"
	c, err = endpoints.GivenEndpoint(context.TODO(), callback2)

	require.NoError(t, err)
	require.Equal(t, callback2, c)

	callback3 := testCall3()
	callback3.Headers[AssuredCallbackKey] = "call-key"
	c, err = endpoints.GivenEndpoint(context.TODO(), callback3)

	require.NoError(t, err)
	require.Equal(t, callback3, c)

	c, err = endpoints.GivenCallbackEndpoint(context.TODO(), testCallback())

	expectedAssured := &ExpectedCallStore{
		data: map[string][]*ExpectedCall{
			"GET:test/assured":    {callback1, callback2},
			"POST:teapot/assured": {callback3},
		},
	}
	expectedCallback := &CallStore{
		data: map[string][]*Call{
			"call-key": {testCallback()},
		},
	}
	require.NoError(t, err)
	require.Equal(t, testCallback(), c)
	require.Equal(t, expectedAssured, endpoints.assuredCalls)
	require.Equal(t, expectedCallback, endpoints.callbackCalls)

}

func TestWhenEndpointSuccess(t *testing.T) {
	endpoints := &AssuredEndpoints{
		assuredCalls:   fullAssuredCalls,
		madeCalls:      NewCallStore(),
		callbackCalls:  NewCallStore(),
		trackMadeCalls: true,
	}
	expected := map[string][]*ExpectedCall{
		"GET:test/assured":    {testCall2(), testCall1()},
		"POST:teapot/assured": {testCall3()},
	}

	c, err := endpoints.WhenEndpoint(context.TODO(), call1)

	require.NoError(t, err)
	require.Equal(t, testCall1(), c)
	require.Equal(t, expected, endpoints.assuredCalls.data)

	c, err = endpoints.WhenEndpoint(context.TODO(), call2)

	require.NoError(t, err)
	require.Equal(t, testCall2(), c)
	require.Equal(t, fullAssuredCalls, endpoints.assuredCalls)

	c, err = endpoints.WhenEndpoint(context.TODO(), call3)

	require.NoError(t, err)
	require.Equal(t, testCall3(), c)
	require.Equal(t, fullAssuredCalls, endpoints.assuredCalls)
	require.Equal(t, fullAssuredCalls, endpoints.madeCalls)
}

func TestWhenEndpointSuccessTrackingDisabled(t *testing.T) {
	endpoints := &AssuredEndpoints{
		assuredCalls:   fullAssuredCalls,
		madeCalls:      NewCallStore(),
		callbackCalls:  NewCallStore(),
		trackMadeCalls: false,
	}
	expected := map[string][]*ExpectedCall{
		"GET:test/assured":    {testCall2(), testCall1()},
		"POST:teapot/assured": {testCall3()},
	}

	c, err := endpoints.WhenEndpoint(context.TODO(), call1)

	require.NoError(t, err)
	require.Equal(t, testCall1(), c)
	require.Equal(t, expected, endpoints.assuredCalls.data)

	c, err = endpoints.WhenEndpoint(context.TODO(), call2)

	require.NoError(t, err)
	require.Equal(t, testCall2(), c)
	require.Equal(t, fullAssuredCalls, endpoints.assuredCalls)

	c, err = endpoints.WhenEndpoint(context.TODO(), call3)

	require.NoError(t, err)
	require.Equal(t, testCall3(), c)
	require.Equal(t, fullAssuredCalls, endpoints.assuredCalls)
	require.Equal(t, NewCallStore(), endpoints.madeCalls)
}

func TestWhenEndpointSuccessCallbacks(t *testing.T) {
	called := false
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	assured := testCall1()
	assured.Headers[AssuredCallbackKey] = "call-key"
	call := testCallback()
	call.Headers[AssuredCallbackTarget] = testServer.URL
	endpoints := &AssuredEndpoints{
		httpClient: http.DefaultClient,
		assuredCalls: &ExpectedCallStore{
			data: map[string][]*ExpectedCall{"GET:test/assured": {assured}},
		},
		madeCalls: NewCallStore(),
		callbackCalls: &CallStore{
			data: map[string][]*Call{"call-key": {call}},
		},
		trackMadeCalls: true,
	}

	c, err := endpoints.WhenEndpoint(context.TODO(), (*Call)(unsafe.Pointer(assured)))

	require.NoError(t, err)
	require.Equal(t, assured, c)
	// allow go routine to finish
	time.Sleep(10 * time.Millisecond)
	require.True(t, called, "callback was not hit")
}

func TestWhenEndpointSuccessDelayed(t *testing.T) {
	called := false
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	assured := testCall1()
	assured.Headers[AssuredCallbackKey] = "call-key"
	assured.Headers[AssuredDelay] = "2"
	call := testCallback()
	call.Headers[AssuredCallbackTarget] = testServer.URL
	call.Headers[AssuredCallbackDelay] = "4"
	endpoints := &AssuredEndpoints{
		httpClient: http.DefaultClient,
		assuredCalls: &ExpectedCallStore{
			data: map[string][]*ExpectedCall{"GET:test/assured": {assured}},
		},
		madeCalls: NewCallStore(),
		callbackCalls: &CallStore{
			data: map[string][]*Call{"call-key": {call}},
		},
		trackMadeCalls: true,
	}
	start := time.Now()
	c, err := endpoints.WhenEndpoint(context.TODO(), (*Call)(unsafe.Pointer(assured)))

	require.True(t, time.Since(start) >= 2*time.Second, "response should be delayed 2 seconds")
	require.NoError(t, err)
	require.Equal(t, assured, c)
	// allow go routine to finish
	time.Sleep(1 * time.Second)
	require.False(t, called, "callback should not be hit yet")
	time.Sleep(2 * time.Second)
	require.True(t, called, "callback was not hit")
}

func TestSendCallbackBadRequest(t *testing.T) {
	called := false
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	call := testCallback()
	call.Method = "\""
	endpoints := NewAssuredEndpoints(DefaultOptions)
	endpoints.sendCallback(testServer.URL, call)

	// allow go routine to finish
	time.Sleep(1 * time.Millisecond)
	require.False(t, called, "callback should not be hit")
}

func TestSendCallbackBadResponse(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)
	endpoints.sendCallback("http://localhost:900000", testCallback())
}

func TestWhenEndpointNotFound(t *testing.T) {
	endpoints := NewAssuredEndpoints(DefaultOptions)

	c, err := endpoints.WhenEndpoint(context.TODO(), call1)

	require.Nil(t, c)
	require.Error(t, err)
	require.Equal(t, "No assured calls", err.Error())
}

func TestVerifyEndpointSuccess(t *testing.T) {
	callStore := convertExpectedCallsToCalls(fullAssuredCalls)

	endpoints := &AssuredEndpoints{
		madeCalls:      callStore,
		trackMadeCalls: true,
	}

	c, err := endpoints.VerifyEndpoint(context.TODO(), call1)

	require.NoError(t, err)
	require.Equal(t, []*Call{call1, call2}, c)

	c, err = endpoints.VerifyEndpoint(context.TODO(), call3)

	require.NoError(t, err)
	require.Equal(t, []*Call{call3}, c)
}

func TestVerifyEndpointTrackingDisabled(t *testing.T) {
	callStore := convertExpectedCallsToCalls(fullAssuredCalls)

	endpoints := &AssuredEndpoints{
		madeCalls:      callStore,
		trackMadeCalls: false,
	}

	c, err := endpoints.VerifyEndpoint(context.TODO(), call1)

	require.Nil(t, c)
	require.Error(t, err)
	require.Equal(t, "Tracking made calls is disabled", err.Error())
}

func TestClearEndpointSuccess(t *testing.T) {
	callStore := convertExpectedCallsToCalls(fullAssuredCalls)

	endpoints := &AssuredEndpoints{
		assuredCalls:   fullAssuredCalls,
		madeCalls:      callStore,
		callbackCalls:  NewCallStore(),
		trackMadeCalls: true,
	}
	expected := map[string][]*Call{
		"POST:teapot/assured": {call3},
	}

	c, err := endpoints.ClearEndpoint(context.TODO(), call1)

	require.NoError(t, err)
	require.Nil(t, c)
	require.Equal(t, expected, endpoints.assuredCalls.data)
	require.Equal(t, expected, endpoints.madeCalls.data)

	c, err = endpoints.ClearEndpoint(context.TODO(), call2)

	require.NoError(t, err)
	require.Nil(t, c)
	require.Equal(t, expected, endpoints.assuredCalls.data)
	require.Equal(t, expected, endpoints.madeCalls.data)

	c, err = endpoints.ClearEndpoint(context.TODO(), call3)

	require.NoError(t, err)
	require.Nil(t, c)
	require.Equal(t, map[string][]*Call{}, endpoints.assuredCalls.data)
	require.Equal(t, map[string][]*Call{}, endpoints.madeCalls.data)
}

func TestClearEndpointSuccessCallback(t *testing.T) {
	endpoints := &AssuredEndpoints{
		assuredCalls: fullAssuredCalls,
		madeCalls:    NewCallStore(),
		callbackCalls: &CallStore{
			data: map[string][]*Call{
				"call-key":       {testCallback()},
				"other-call-key": {testCallback()},
			},
		},
		trackMadeCalls: true,
	}

	c, err := endpoints.ClearEndpoint(context.TODO(), testCallback())

	require.NoError(t, err)
	require.Nil(t, c)
	require.Equal(t, fullAssuredCalls.data, endpoints.assuredCalls.data)
	require.Equal(t, map[string][]*Call{}, endpoints.madeCalls.data)
	require.Equal(t, map[string][]*Call{"other-call-key": {testCallback()}}, endpoints.callbackCalls.data)
}

func TestClearAllEndpointSuccess(t *testing.T) {
	callStore := convertExpectedCallsToCalls(fullAssuredCalls)

	endpoints := &AssuredEndpoints{
		assuredCalls:   fullAssuredCalls,
		madeCalls:      callStore,
		callbackCalls:  callStore,
		trackMadeCalls: true,
	}

	c, err := endpoints.ClearAllEndpoint(context.TODO(), nil)

	require.NoError(t, err)
	require.Nil(t, c)
	require.Equal(t, map[string][]*Call{}, endpoints.assuredCalls.data)
	require.Equal(t, map[string][]*Call{}, endpoints.madeCalls.data)
	require.Equal(t, map[string][]*Call{}, endpoints.callbackCalls.data)
}
