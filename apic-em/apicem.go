package apicem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "0.1.0"
	userAgent      = "apicem/" + libraryVersion
	mediaType      = "application/json"
)

var (
	defaultBaseURL     = "https://sandboxapic.cisco.com/api"
	authorizationToken = ""
)

// Client manages communication with Cisco Spark V1 API.
type Client struct {
	// HTTP client used to communicate with the Cisco Spark API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Authorization is the authentication token
	Authorization string

	common service // Reuse a single struct instead of allocating one for each service on the heap

	// Services used for communicating with the APIC-EM API
	AAA                   *AAAService
	Alarm                 *AlarmService
	Application           *ApplicationService
	Audit                 *AuditService
	Category              *CategoryService
	CertificateManagement *CertificateManagementService
	CiscoISE              *CiscoISEService
	GlobalCredential      *GlobalCredentialService
	Discovery             *DiscoveryService
	// FileService           *FileServiceService
	FlowAnalysis        *FlowAnalysisService
	Host                *HostService
	Interface           *InterfaceService
	IPGeo               *IPGeoService
	IPPool              *IPPoolService
	License             *LicenseService
	Location            *LocationService
	Neighborhood        *NeighborhoodService
	NetworkDevice       *NetworkDeviceService
	NetworkDeviceConfig *NetworkDeviceConfigService
	PKIBroker           *PKIBrokerService
	Policy              *PolicyService
	ReachabilityInfo    *ReachabilityInfoService
	Relevance           *RelevanceService
	Role                *RoleService
	ScalableGroup       *ScalableGroupService
	Scheduler           *SchedulerService
	Segment             *SegmentService
	Tag                 *TagService
	Task                *TaskService
	Ticket              *TicketService
	Topology            *TopologyService
	TopologyApplication *TopologyApplicationService
	TopologyVLAN        *TopologyVLANService
	User                *UserService
	Contract            *ContractService
	PolicyV2            *PolicyV2Service
	VLAN                *VLANService
	VRF                 *VRFService

	// Optional function called after every successful request made to the Cisco Spark APIs
	onRequestCompleted RequestCompletionCallback
}

type service struct {
	client *Client
}

// RequestCompletionCallback defines the type of the request callback function
type RequestCompletionCallback func(*http.Request, *http.Response)

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	// For paginated result sets, page of results to retrieve.
	Page int `url:"page,omitempty"`

	// For paginated result sets, the number of results to include per page.
	PerPage int `url:"per_page,omitempty"`
}

// Response is a Cisco Spark response. This wraps the standard http.Response returned from Cisco Spark.
type Response struct {
	*http.Response

	// Monitoring URI
	Monitor string
}

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {
	// HTTP response that caused this error
	HTTPResponse *http.Response

	Message string
	Errors  []struct {
		Description string
	}
	TrackingID string
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)

	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	origURL, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	origValues := origURL.Query()

	newValues, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	for k, v := range newValues {
		origValues[k] = v
	}

	origURL.RawQuery = origValues.Encode()
	return origURL.String(), nil
}

// NewClient returns a new Cisco Spark API client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL, _ := url.Parse(defaultBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent, Authorization: authorizationToken}
	c.common.client = c
	c.AAA = (*AAAService)(&c.common)
	c.Alarm = (*AlarmService)(&c.common)
	c.Application = (*ApplicationService)(&c.common)
	c.Audit = (*AuditService)(&c.common)
	c.Category = (*CategoryService)(&c.common)
	c.CertificateManagement = (*CertificateManagementService)(&c.common)
	c.CiscoISE = (*CiscoISEService)(&c.common)
	c.GlobalCredential = (*GlobalCredentialService)(&c.common)
	c.Discovery = (*DiscoveryService)(&c.common)
	// c.FileService = (*FileServiceService)(&c.common)
	c.FlowAnalysis = (*FlowAnalysisService)(&c.common)
	c.Host = (*HostService)(&c.common)
	c.Interface = (*InterfaceService)(&c.common)
	c.IPGeo = (*IPGeoService)(&c.common)
	c.IPPool = (*IPPoolService)(&c.common)
	c.License = (*LicenseService)(&c.common)
	c.Location = (*LocationService)(&c.common)
	c.Neighborhood = (*NeighborhoodService)(&c.common)
	c.NetworkDevice = (*NetworkDeviceService)(&c.common)
	c.NetworkDeviceConfig = (*NetworkDeviceConfigService)(&c.common)
	c.PKIBroker = (*PKIBrokerService)(&c.common)
	c.Policy = (*PolicyService)(&c.common)
	c.ReachabilityInfo = (*ReachabilityInfoService)(&c.common)
	c.Relevance = (*RelevanceService)(&c.common)
	c.Role = (*RoleService)(&c.common)
	c.ScalableGroup = (*ScalableGroupService)(&c.common)
	c.Scheduler = (*SchedulerService)(&c.common)
	c.Segment = (*SegmentService)(&c.common)
	c.Tag = (*TagService)(&c.common)
	c.Task = (*TaskService)(&c.common)
	c.Ticket = (*TicketService)(&c.common)
	c.Topology = (*TopologyService)(&c.common)
	c.TopologyApplication = (*TopologyApplicationService)(&c.common)
	c.TopologyVLAN = (*TopologyVLANService)(&c.common)
	c.User = (*UserService)(&c.common)
	c.Contract = (*ContractService)(&c.common)
	c.PolicyV2 = (*PolicyV2Service)(&c.common)
	c.VLAN = (*VLANService)(&c.common)
	c.VRF = (*VRFService)(&c.common)
	return c
}

// ClientOpt are options for New.
type ClientOpt func(*Client) error

// New returns a new Cisco Spark API client instance.
func New(httpClient *http.Client, opts ...ClientOpt) (*Client, error) {
	c := NewClient(httpClient)
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// SetBaseURL is a client option for setting the base URL.
func SetBaseURL(bu string) ClientOpt {
	return func(c *Client) error {
		u, err := url.Parse(bu)
		if err != nil {
			return err
		}

		c.BaseURL = u
		return nil
	}
}

// SetUserAgent is a client option for setting the user agent.
func SetUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.UserAgent = fmt.Sprintf("%s+%s", ua, c.UserAgent)
		return nil
	}
}

// NewRequest creates an API request. A relative URL can be provided in urlStr, which will be resolved to the
// BaseURL of the Client. Relative URLS should always be specified without a preceding slash. If specified, the
// value pointed to by body is JSON encoded and included in as the request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	method = strings.ToUpper(method)
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	buf := new(bytes.Buffer)
	if body != nil {
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mediaType)
	req.Header.Add("Accept", mediaType)
	req.Header.Add("User-Agent", c.UserAgent)
	if c.Authorization != "" {
		req.Header.Add("X-Auth-Token", c.Authorization)
	}
	return req, nil
}

// OnRequestCompleted sets the Cisco Spark API request completion callback
func (c *Client) OnRequestCompleted(rc RequestCompletionCallback) {
	c.onRequestCompleted = rc
}

// newResponse creates a new Response for the provided http.Response
func newResponse(r *http.Response) *Response {
	response := Response{Response: r}

	return &response
}

// Do sends an API request and returns the API response. The API response is JSON decoded and stored in the value
// pointed to by v, or returned as an error if an API error has occurred. If v implements the io.Writer interface,
// the raw response will be written to v, without attempting to decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if c.onRequestCompleted != nil {
		c.onRequestCompleted(req, resp)
	}

	defer func() {
		if rerr := resp.Body.Close(); err == nil {
			err = rerr
		}
	}()

	defer func() {
		// Drain up to 512 bytes and close the body to let the Transport reuse the connection
		io.CopyN(ioutil.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err := io.Copy(w, resp.Body)
			if err != nil {
				return nil, err
			}
		} else {
			err := json.NewDecoder(resp.Body).Decode(v)
			if err != nil {
				return nil, err
			}
		}
	}

	return response, err
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v",
		r.HTTPResponse.Request.Method, r.HTTPResponse.Request.URL, r.HTTPResponse.StatusCode, r.Message)
}

// CheckResponse checks the API response for errors, and returns them if present. A response is considered an
// error if it has a status code outside the 200 range. API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other response body will be silently ignored.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{HTTPResponse: r}
	fmt.Println("ERROR", errorResponse)

	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			return err
		}
	}

	return errorResponse
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

// StreamToString converts a reader to a string
func StreamToString(stream io.Reader) string {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(stream)
	return buf.String()
}
