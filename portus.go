package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

const (
	KEYMANAGER_ENDPOINT  = "key-manager"
	PORTUS_KEY_FILENAME  = "test_portus_key"
	PORTUS_CERT_FILENAME = "test_portus_crt"
)

var PortusSelectedNodeId uint64

func AddPortus(t *testing.T) {
	t.Run("addPortus", installPortus)
}

func CheckPortusInstallation(t *testing.T) {
	t.Run("checkPortus", checkPortus)
}

func UploadSecurityPortusKey(t *testing.T) {
	t.Run("uploadPortusKey", uploadSecurityKey_Portus)
}

func UploadSecurityPortusCert(t *testing.T) {
	t.Run("uploadSecurityPortusCert", uploadCertificate_Portus)
}

func uploadSecurityKey_Portus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := CreateFileAndUpload(PORTUS_KEY_FILENAME, PORTUS_KEY, PRIVATE_KEY)
	if err != nil {
		assert.Fatalf(err.Error())
	}
}

func uploadCertificate_Portus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := CreateFileAndUpload(PORTUS_CERT_FILENAME, PORTUS_CERT, CERT)
	if err != nil {
		assert.Fatalf(err.Error())
	}
}

func installPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		portusConfiguration pcc.PortusConfiguration
	)

	for id, node := range Nodes {
		if !IsInvader(node) && Pcc.IsNodeOnline(node) {
			portusConfiguration = Env.PortusConfiguration
			portusConfiguration.NodeID = id
			portusConfiguration.Name = fmt.Sprintf("portus_%v", id)

			if Env.AuthenticationProfile.Name == "" {
				fmt.Println("Authenticatiom Profile is not defined in the configuration file, Portus will be installed without it")
			} else {
				authProfile, err := Pcc.GetAuthProfileByName(CurrentAuthProfileName)
				if err == nil {
					data, err := json.Marshal(authProfile)
					if err != nil {
						assert.Fatalf("marshal failed\n")
						return
					}
					err = json.Unmarshal(data, &portusConfiguration.AuthenticationProfile)
					if err != nil {
						assert.Fatalf("unmarshal failed\n")
						return
					}
				} else {
					fmt.Printf("Missing authentication profile %s\n, Portus will be installed without it", CurrentAuthProfileName)
				}
			}

			certificate, err := Pcc.FindCertificate(PORTUS_CERT_FILENAME)
			if err != nil {
				fmt.Printf("Get certificate %s failed\n%v\n", PORTUS_CERT_FILENAME, err)
			} else {
				portusConfiguration.RegistryCertId = &certificate.Id
			}

			privateKey, err := Pcc.FindSecurityKey(PORTUS_KEY_FILENAME)
			if err != nil {
				fmt.Printf("Get private key %s failed\n%v\n", PORTUS_KEY_FILENAME, err)
			} else {
				portusConfiguration.RegistryKeyId = &privateKey.Id
			}

			fmt.Printf("Installing Portus on Node with id %v\n",
				node.Id)

			err = Pcc.InstallPortusNode(portusConfiguration)
			if err != nil {
				assert.Fatalf("Failed to install Portus: %v\n",
					err)
				return
			}

			PortusSelectedNodeId = node.Id
			break
		}

	}
}

func checkPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	from := time.Now()

	for id, node := range Nodes {
		if !IsInvader(node) && Pcc.IsNodeOnline(node) && node.Id == PortusSelectedNodeId {
			check, err := checkGenericInstallation(id, PORTUS_TIMEOUT, PORTUS_NOTIFICATION, from)
			if err != nil {
				assert.Fatalf("Portus installation has failed\n%v\n", err)
			}
			if check {
				fmt.Printf("Portus correctly installed on nodeId:%v\n", id)
			}
		}
	}
}

func delAllPortus(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		portusConfigs []pcc.PortusConfiguration
		err           error
		id            uint64
	)

	portusConfigs, err = Pcc.GetPortusNodes()
	if err != nil {
		assert.Fatalf("Failed to get portus nodes: %v\n", err)
		return
	}

	for _, p := range portusConfigs {
		fmt.Printf("Deleting Portus %v\n", p.Name)
		id = p.ID
		err = Pcc.DelPortusNode(id, true)
		if err != nil {
			assert.Fatalf("Failed to delete Portus %v: %v\n",
				p.Name, err)
			return
		}
		// wait till deleted
		done := false
		timeout := time.After(10 * time.Minute)
		tick := time.Tick(30 * time.Second)
		for !done {
			select {
			case <-tick:
				_, err = Pcc.GetPortusNodeById(id)
				if err != nil {
					assert.Fatalf("Failed Get Portus: %v\n",
						err)
					return
				}
				fmt.Printf("get portus by id err: %v\n", err)
				if err.Error() == "record not found" {
					done = true
					continue
				} else {
					fmt.Printf("get portus by id err: %v\n",
						err)
				}
			case <-timeout:
				assert.Fatal("Timeout deleting Portus\n")
			}
		}
	}
}
