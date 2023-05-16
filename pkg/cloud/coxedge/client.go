package coxedge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	baseURLDefault = "https://portal.coxedge.com/api/v1/"
)

var (
	ErrWorkloadNotFound = errors.New("workload not found")
)

type Client struct {
	client         *http.Client
	apiKey         string
	baseURL        *url.URL
	service        string
	environment    string
	organizationID string
}

func NewClient(baseURL, service, environment, apiKey string, organizationID string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	if baseURL == "" {
		baseURL = baseURLDefault
	}

	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	client := &Client{
		apiKey:  apiKey,
		baseURL: url,
		client:  httpClient,
	}

	if organizationID != "" {
		client.organizationID = fmt.Sprintf("org_id=%s", organizationID)
	} else {
		client.organizationID = ""
	}
	client.service = service
	client.environment = environment

	return client, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads/549ec584-c62b-4647-9ca0-f04f9a88403d'
func (c *Client) GetWorkload(id string) (*Workload, error) {
	w := &Workload{}
	err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/workloads/%s?%s", c.service, c.environment, id, c.organizationID), nil, w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (c *Client) GetWorkloadByName(name string) (*WorkloadData, error) {
	workloads, err := c.GetWorkloads()
	if err != nil {
		return nil, err
	}
	name = shortenName(name, 18)

	for _, workload := range workloads.Data {
		if workload.Name == name {
			return &workload, nil
		}
	}
	return nil, ErrWorkloadNotFound
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads'
func (c *Client) GetWorkloads() (*Workloads, error) {
	w := &Workloads{}
	err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/workloads?%s", c.service, c.environment, c.organizationID), nil, w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/instances?workloadId=5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d' | jq
func (c *Client) GetInstances(workloadID string) (*Instances, error) {
	i := &Instances{}
	err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/instances?workloadId=%s&%s", c.service, c.environment, workloadID, c.organizationID), nil, i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// curl -X 'GET' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/instances/5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d/capi-test-jg90-wi-peter-qhl-waw-0' | jq
func (c *Client) GetInstance(instanceID string) (*Instance, error) {
	i := &Instance{}
	err := c.DoRequest("GET", fmt.Sprintf("/services/%s/%s/instances/%s?%s", c.service, c.environment, instanceID, c.organizationID), nil, i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (c *Client) GetTask(taskID string) (*Task, error) {
	t := &Task{}
	err := c.DoRequest("GET", fmt.Sprintf("/tasks/%s?%s", taskID, c.organizationID), nil, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (c *Client) WaitForWorkload(taskID string) (string, error) {
	t, err := c.GetTask(taskID)
	if err != nil {
		return "", err
	}
	switch t.Data.Status {
	case "SUCCESS":
		return t.Data.Result.ID, nil
	case "FAILURE":
		return "", fmt.Errorf("provisioning of workload failed")
	default:
		time.Sleep(5 * time.Second)
		return c.WaitForWorkload(taskID)
	}
}

// curl -X 'POST' -d '{"name":"capi-test-jg90","type":"VM","image":"stackpath-edge/centos-7:v202103021226","addAnyCastIpAddress":true,"ports":[{"protocol":"TCP","publicPort":"22"},{"protocol":"TCP","publicPort":"80"}],"firstBootSshKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDgnV5MOhBqpQLt66KGlMKi/VYtmVPUt6epSVxnxrvjayNto5flG2sH4cGqdI2C0NE9/w7BFNdwWqp0mL2kYynC8l+SejW/qjx37hrEBWIXqdTyumchm0LD/7K7P7/kz14IV5NcHjNAsntPgKjx/fzJlbA1VCQYmnOq9RZeKme44rdHYW0BBfgMzekcEbyGTNDGp51NYhVafZLXsF8MzCKlJ+NCPlDqzD6w0fQe/qtMFO8NbFyS9/Lk4prp4HAWEyLSM26w1iLycYpbpWrHw6oc1U7bNIgbsa0ezDu4+OPkxeHz7aG5TeJ/dn0Wftzdfy2sy5PJy5MnYP3RTuROsOv+chu+AshZNNJ9A4ar5gFXSX40sQ0i4GzxZGrsKhW42ZP4sElzV74gEBQ2BOIOJUh4qGRtnjsQCJHBs7DLgpeVeGUq2B7p5zDAlJBGCXiHuTgIM8aVnpdnNrFwmr9SF66iaTrt7x8HinNOCIIztMU15Fk2AYSxSEuju1d3VcPt/d0= jasmingacic@Jasmins-MBP","deployments":[{"name":"wi-peter-qhl","pops":["WAW"],"instancesPerPop":"1"}],"specs":"SP-5"}' -H 'Mc-Api-Key: $TOKEN' 'https://portal.coxedge.com/api/v1/services/edge-services/faefawef/workloads'
func (c *Client) CreateWorkload(data *CreateWorkloadRequest) (*POSTResponse, error) {
	pr := &POSTResponse{}
	data.Name = shortenName(data.Name, 18)

	err := c.DoRequest("POST", fmt.Sprintf("/services/%s/%s/workloads?%s", c.service, c.environment, c.organizationID), data, pr)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (c *Client) DeleteWorkload(workloadID string) (*POSTResponse, error) {
	pr := &POSTResponse{}
	wl, err := c.GetWorkload(workloadID)
	if err != nil {
		return nil, err
	}

	err = c.DoRequest("POST", fmt.Sprintf("/services/%s/%s/workloads/%s?operation=delete&%s", c.service, c.environment, workloadID, c.organizationID), wl.Data, pr)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (c *Client) UpdateWorkload(workloadID string, workload WorkloadData) (*POSTResponse, error) {
	pr := &POSTResponse{}
	workload.Name = shortenName(workload.Name, 18)

	err := c.DoRequest("PUT", fmt.Sprintf("/services/%s/%s/workloads/%s?%s", c.service, c.environment, workloadID, c.organizationID), workload, pr)
	if err != nil {
		return nil, err
	}

	return pr, err
}

func (c *Client) DoRequest(method, path string, body, v interface{}) error {
	req, err := c.NewRequest(method, path, body)
	if err != nil {
		return err
	}

	return c.Do(req, v)
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	// relative path to append to the endpoint url, no leading slash please
	if path[0] == '/' {
		path = path[1:]
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(rel)
	var req *http.Request
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, u.String(), bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	}
	req.Header.Add("MC-Api-Key", c.apiKey)
	req.Close = true

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		o, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(o),
		}
	}
	o, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(o, v)
}

func shortenName(name string, limit int) string {
	if len(name) <= limit {
		return name
	}

	parts := strings.Split(name, "-")
	var postfix string
	var resize string
	if len(parts) <= 1 {
		resize = strings.Join(parts, "-")
	} else {
		postfix = "-" + parts[len(parts)-1]
		resize = strings.Join(parts[:len(parts)-1], "-")
	}

	trimRange := len(name) - limit
	return resize[:len(resize)-trimRange] + postfix
}

type Workload struct {
	Data WorkloadData `json:"data,omitempty"`
}

type Workloads struct {
	Data []WorkloadData `json:"data,omitempty"`
}
type WorkloadData struct {
	ID                            string                `json:"id"`
	Name                          string                `json:"name"`
	StackID                       string                `json:"stackId"`
	Slug                          string                `json:"slug"`
	Version                       string                `json:"version"`
	Type                          string                `json:"type"`
	Network                       string                `json:"network"`
	CPU                           string                `json:"cpu"`
	Memory                        string                `json:"memory"`
	IsRemoteManagementEnabled     bool                  `json:"isRemoteManagementEnabled"`
	Image                         string                `json:"image"`
	AddImagePullCredentialsOption bool                  `json:"addImagePullCredentialsOption"`
	EnvironmentVariable           []EnvironmentVariable `json:"environmentVariables"`
	SecretEnvironmentVariables    []EnvironmentVariable `json:"secretEnvironmentVariables"`
	AddAnyCastIPAddress           bool                  `json:"addAnyCastIpAddress"`
	AnycastIPAddress              string                `json:"anycastIpAddress"`
	FirstBootSSHKey               string                `json:"firstBootSshKey"`
	Specs                         string                `json:"specs"`
	Deployments                   []Deployment          `json:"deployments"`
	Status                        string                `json:"status"`
	Created                       time.Time             `json:"created"`
	Ports                         []Port                `json:"ports"`
	PersistenceStorageTotalSize   int                   `json:"persistenceStorageTotalSize,omitempty"`
	NetworkInterfaces             []NetworkInterface    `json:"networkInterfaces"`
}

type Instances struct {
	Data     []InstanceData `json:"data,omitempty"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

type Instance struct {
	Data InstanceData `json:"data,omitempty"`
}

type CreateWorkloadRequest struct {
	Name                          string                `json:"name,omitempty"`
	Type                          string                `json:"type,omitempty"`
	Image                         string                `json:"image,omitempty"`
	AddImagePullCredentialsOption bool                  `json:"addImagePullCredentialsOption,omitempty"`
	EnvironmentVariables          []EnvironmentVariable `json:"environmentVariables,omitempty"`
	SecretEnvironmentVariables    []EnvironmentVariable `json:"secretEnvironmentVariables,omitempty"`
	AddAnyCastIPAddress           bool                  `json:"addAnyCastIpAddress,omitempty"`
	PersistenceStorageTotalSize   int                   `json:"persistenceStorageTotalSize"`
	Ports                         []Port                `json:"ports,omitempty"`
	FirstBootSSHKey               string                `json:"firstBootSshKey"`
	Deployments                   []Deployment          `json:"deployments,omitempty"`
	Specs                         string                `json:"specs,omitempty"`
	PersistentStorages            []PersistentStorage   `json:"persistentStorages,omitempty"`
	ContainerUsername             string                `json:"containerUsername,omitempty"`
	ContainerPassword             string                `json:"containerPassword,omitempty"`
	ContainerServer               string                `json:"containerServer,omitempty"`
	Commands                      []string              `json:"commands,omitempty"`
	UserData                      string                `json:"userData,omitempty"`
	NetworkInterfaces             []NetworkInterface    `json:"networkInterfaces"`
}

type POSTResponse struct {
	TaskID     string `json:"TaskID,omitempty"`
	TaskStatus string `json:"taskStatus,omitempty"`
}

type Port struct {
	Protocol       string `json:"protocol"`
	PublicPort     string `json:"publicPort"`
	PublicPortDesc string `json:"publicPortDesc,omitempty"`
}

type EnvironmentVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Deployment defines instance specifications
type Deployment struct {
	// Name of the deployment instance
	Name string `json:"name,omitempty"`
	// CoxEdge PoPs - geographical location for the instance
	Pops []string `json:"pops,omitempty"`
	// +optional
	EnableAutoScaling bool `json:"enableAutoScaling,omitempty"`
	// number of instances per each PoP defined
	// +optional
	InstancesPerPop string `json:"instancesPerPop,omitempty"`
	// +optional
	CPUUtilization int `json:"cpuUtilization,omitempty"`
	// +optional
	MinInstancesPerPop string `json:"minInstancesPerPop,omitempty"`
	// +optional
	MaxInstancesPerPop string `json:"maxInstancesPerPop,omitempty"`
}

type InstanceData struct {
	StackID                   string       `json:"stackId"`
	WorkloadID                string       `json:"workloadId"`
	WorkloadName              string       `json:"workloadName"`
	Name                      string       `json:"name"`
	Type                      string       `json:"type"`
	IPAddress                 []string     `json:"ipAddress"`
	PublicIPAddress           string       `json:"publicIpAddress"`
	Location                  string       `json:"location"`
	Created                   time.Time    `json:"created"`
	StartedDate               time.Time    `json:"startedDate"`
	Image                     string       `json:"image"`
	CPU                       string       `json:"cpu"`
	Memory                    string       `json:"memory"`
	EphemeralStorageSize      string       `json:"ephemeralStorageSize"`
	Version                   string       `json:"version"`
	IsRemoteManagementEnabled bool         `json:"isRemoteManagementEnabled"`
	LocationInfo              LocationInfo `json:"locationInfo"`
	InstanceKeyName           string       `json:"instanceKeyName"`
	ID                        string       `json:"id"`
	Status                    string       `json:"status"`
}

type LocationInfo struct {
	City            string  `json:"city"`
	CityCode        string  `json:"cityCode"`
	Subdivision     string  `json:"subdivision"`
	SubdivisionCode string  `json:"subdivisionCode"`
	Country         string  `json:"country"`
	CountryCode     string  `json:"countryCode"`
	Continent       string  `json:"continent"`
	Latitude        float64 `json:"latitude"`
}

type Metadata struct {
	RecordCount int `json:"recordCount"`
}

type HTTPError struct {
	StatusCode int
	Message    string
}

var _ error = (*HTTPError)(nil)

func (e *HTTPError) Error() string {
	return fmt.Sprintf("coxedge http client: %s (%d)", e.Message, e.StatusCode)
}

type Task struct {
	Data struct {
		ID      string    `json:"id"`
		Status  string    `json:"status"`
		Created time.Time `json:"created"`
		Result  struct {
			PortRange       string `json:"portRange"`
			Protocol        string `json:"protocol"`
			StackID         string `json:"stackId"`
			WorkloadID      string `json:"workloadId"`
			Description     string `json:"description"`
			Action          string `json:"action"`
			ID              string `json:"id"`
			Source          string `json:"source"`
			Type            string `json:"type"`
			NetworkPolicyID string `json:"networkPolicyId"`
		} `json:"result"`
	} `json:"data"`
}

type PersistentStorage struct {
	Path string `json:"path"`
	Size string `json:"size"`
}

type NetworkInterface struct {
	VPCSlug    string `json:"vpcSlug"`
	IPFamilies string `json:"ipFamilies"`
	SubnetSlug string `json:"subnetSlug"`
	IsPublicIP bool   `json:"isPublicIP"`
}
