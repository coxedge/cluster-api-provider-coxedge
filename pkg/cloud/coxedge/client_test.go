package coxedge

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

var (
	wlID string
	c    *Client
	skip bool
)

func init() {
	if len(os.Getenv("COX_SKIP_TESTS")) > 0 {
		skip = true
	}

	c, _ = NewClient(os.Getenv("COXEDGE_SERVICE"), os.Getenv("COXEDGE_ENVIRONMENT"), os.Getenv("COXEDGE_TOKEN"), nil)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestCreateWorkload(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	workload := &CreateWorkloadRequest{
		Name:                        "test-capi-cox-" + RandStringRunes(4),
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
				Name:              "test-capi-cox",
				EnableAutoScaling: false,
				InstancesPerPop:   "1",
				CPUUtilization:    0,
				Pops:              []string{"WAW"},
			},
		},
		Specs: "SP-5",
	}
	pr, resp, err := c.CreateWorkload(workload)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
	wlID, err = c.WaitForWorkload(pr.TaskID)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
}

func TestGetWorkloads(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	_, resp, err := c.GetWorkloads()

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
}

func TestGetWorkload(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	wl, resp, err := c.GetWorkload(wlID)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}

	if wl.Data.ID != wlID {
		t.Logf("fetched data is not equal to sought data wanted: %s, received: %s ", wlID, wl.Data.ID)
	}
}

func TestGetInstances(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	_, resp, err := c.GetInstances(wlID)

	if err != nil {
		t.Log(resp)
		t.Log(err)
		t.Fail()
		return
	}
}

func TestUpdate(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	wl, _, _ := c.GetWorkload(wlID)

	if wl == nil {
		t.Fail()
		return
	}

	data := wl.Data
	data.EnvironmentVariable = append(data.EnvironmentVariable, EnvironmentVariable{Key: "JASMIN", Value: "jasmin"})
	r, _, err := c.UpdateWorkload(wl.Data.ID, data)
	t.Log(r)
	t.Log(err)
}

func TestDeleteWorkload(t *testing.T) {
	if skip {
		t.Skip("COX_SKIP_TESTS is set. Skipping!!!")
	}
	tt, r, err := c.DeleteWorkload(wlID)
	t.Log(tt)
	t.Log(r)
	t.Log(err)
}
