package coxedge

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

const (
	service     = "edge-services"
	environment = "faefawef"
)

func TestCreateWorkload(t *testing.T) {
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)

	workload := &CreateWorkloadRequest{
		Name:                        "testk0s",
		Type:                        "VM",
		Image:                       "stackpath-edge/centos-7:v202103021226",
		AddAnyCastIPAddress:         true,
		PersistenceStorageTotalSize: 0,
		Ports: []Port{
			{
				Protocol:   "TCP",
				PublicPort: "6443",
			},
			{
				Protocol:   "TCP",
				PublicPort: "22",
			},
			{
				Protocol:   "TCP",
				PublicPort: "80",
			},
		},
		FirstBootSSHKey: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDgnV5MOhBqpQLt66KGlMKi/VYtmVPUt6epSVxnxrvjayNto5flG2sH4cGqdI2C0NE9/w7BFNdwWqp0mL2kYynC8l+SejW/qjx37hrEBWIXqdTyumchm0LD/7K7P7/kz14IV5NcHjNAsntPgKjx/fzJlbA1VCQYmnOq9RZeKme44rdHYW0BBfgMzekcEbyGTNDGp51NYhVafZLXsF8MzCKlJ+NCPlDqzD6w0fQe/qtMFO8NbFyS9/Lk4prp4HAWEyLSM26w1iLycYpbpWrHw6oc1U7bNIgbsa0ezDu4+OPkxeHz7aG5TeJ/dn0Wftzdfy2sy5PJy5MnYP3RTuROsOv+chu+AshZNNJ9A4ar5gFXSX40sQ0i4GzxZGrsKhW42ZP4sElzV74gEBQ2BOIOJUh4qGRtnjsQCJHBs7DLgpeVeGUq2B7p5zDAlJBGCXiHuTgIM8aVnpdnNrFwmr9SF66iaTrt7x8HinNOCIIztMU15Fk2AYSxSEuju1d3VcPt/d0= jasmingacic@Jasmins-MBP`,
		Deployments: []Deployment{
			{
				Name:              "testk0s",
				EnableAutoScaling: false,
				InstancesPerPop:   "1",
				CPUUtilization:    0,
				Pops:              []string{"WAW"},
			},
		},
		Specs: "SP-5",
		UserData: `curl -sSLf https://get.k0s.sh | sudo sh
sudo /usr/local/bin/k0s insatll controller --single
sudo /usr/local/bin/k0s start`,
	}
	pr, resp, err := c.CreateWorkload(workload)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
	_, err = c.WaitForWorkload(pr.TaskId)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
}

func TestGetWorkloads(t *testing.T) {
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)
	_, resp, err := c.GetWorkloads()

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}

}

func TestGetWorkload(t *testing.T) {
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)
	id := "ccc07bfe-3647-4417-84e6-2ddb70f2878b"
	wl, resp, err := c.GetWorkload(id)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}

	t.Log(spew.Sdump(wl))
}

func TestGetInstances(t *testing.T) {
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)
	wlid := "5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d"
	instances, resp, err := c.GetInstances(wlid)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
	t.Log(spew.Sdump(instances))
}

//
func TestGetInstance(t *testing.T) {
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)
	id := "5e1eb085-e9b3-447b-8a0e-c0147fc0ea4d/capi-test-jg90-wi-peter-qhl-waw-0"
	instance, resp, err := c.GetInstance(id)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
	t.Log(spew.Sdump(instance))
}

func TestDeleteWorkload(t *testing.T) {
	t.Fail()
	c, _ := NewClient(service, environment, os.Getenv("COXEDGE_TOKEN"), nil)
	tt, r, err := c.DeleteWorkload("36fac36d-8bf8-4ab4-8be1-4c4c7cb65da6")
	t.Log(tt)
	t.Log(r)
	t.Log(err)
}

func TestSomething(t *testing.T) {
	sm := `{"name":"testk0s","stackId":"58191cc3-fe0d-4fd6-9557-28f1cffd1900","slug":"testk0s","version":"1","created":"2021-11-30T19:52:38.388831514Z","type":"VM","network":"default","cpu":"8","memory":"32Gi","isRemoteManagementEnabled":false,"image":"stackpath-edge/centos-7:v202103021226","addImagePullCredentialsOption":false,"environmentVariables":[],"secretEnvironmentVariables":[],"addAnyCastIpAddress":true,"anycastIpAddress":"185.85.196.18","firstBootSshKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDgnV5MOhBqpQLt66KGlMKi/VYtmVPUt6epSVxnxrvjayNto5flG2sH4cGqdI2C0NE9/w7BFNdwWqp0mL2kYynC8l+SejW/qjx37hrEBWIXqdTyumchm0LD/7K7P7/kz14IV5NcHjNAsntPgKjx/fzJlbA1VCQYmnOq9RZeKme44rdHYW0BBfgMzekcEbyGTNDGp51NYhVafZLXsF8MzCKlJ+NCPlDqzD6w0fQe/qtMFO8NbFyS9/Lk4prp4HAWEyLSM26w1iLycYpbpWrHw6oc1U7bNIgbsa0ezDu4+OPkxeHz7aG5TeJ/dn0Wftzdfy2sy5PJy5MnYP3RTuROsOv+chu+AshZNNJ9A4ar5gFXSX40sQ0i4GzxZGrsKhW42ZP4sElzV74gEBQ2BOIOJUh4qGRtnjsQCJHBs7DLgpeVeGUq2B7p5zDAlJBGCXiHuTgIM8aVnpdnNrFwmr9SF66iaTrt7x8HinNOCIIztMU15Fk2AYSxSEuju1d3VcPt/d0= jasmingacic@Jasmins-MBP","specs":"SP-5","persistenceStorageTotalSize":0,"userData":"curl -sSLf https://get.k0s.sh | sudo sh\nsudo /usr/local/bin/k0s insatll controller --single\nsudo /usr/local/bin/k0s start","deployments":[{"name":"testk0s","pops":["WAW"],"enableAutoScaling":false,"instancesPerPop":"1","cpuUtilization":0,"popNames":["Warsaw"]}],"secretEnvVarKeys":[],"id":"b17dd1cd-c50d-464f-9497-71b44b35ef89","status":"ACTIVE"}`
	wl := Workload{}
	if err := json.Unmarshal([]byte(sm), &wl); err != nil {
		t.Fail()
		t.Log(err)
	}

}
