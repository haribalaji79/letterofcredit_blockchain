/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type LoginInfo struct {
	Username	string	`json:"userName"`
	Password	string	`json:"password"`
}

type User struct {
	Username 		string `json:"userName"`
	Role		 	string `json:"role"`
	Password		string `json:"password"`
}

type LC struct {
	ShipmentId				string	  `json:"shipmentId"`
	ContentDescription		string	  `json:"contentDesc"`
	ContentValue			float32	  `json:"contentValue"`
	ExporterCompany			string	  `json:"exporterCompany"`
	ExporterBank			string	  `json:"exporterBank"`
	ImporterCompany			string	  `json:"importerCompany"`
	ImporterBank			string	  `json:"importerBank"`
	FreightCompany			string	  `json:"freightCompany"`
	PortOfLoading			string	  `json:"portOfLoading"`
	PortOfEntry				string	  `json:"portOfEntry"`
	CurrentStatus			string	  `json:"currentStatus"`	
	DocumentNames			[]string  `json:"documentNames"`
	ExporterBankApproved	bool	  `json:"exporterBankApproved"`
	ExporterDocsUploaded	bool	  `json:"exporterDocsUploaded"`
	CustomsApproved			bool	  `json:"customsApproved"`
	PaymentComplete			bool	  `json:"paymentComplete"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("Init firing. Function will be ignored: " + function)

	// Initialize the collection of commercial paper keys
	fmt.Println("Initializing user accounts")
	t.createUser(stub, []string{"importerBank", "importerBank", "Importer Bank"})
	t.createUser(stub, []string{"customs", "customs", "Customs"})
	t.createUser(stub, []string{"exporterBank", "exporterBank", "Exporter Bank"})
	t.createUser(stub, []string{"exporter", "exporter", "Exporter"})
	
	fmt.Println("Initializing LC keys collection if not present")
	valAsbytes, err := stub.GetState("LCKeys")
	if err == nil {
		var keys []string
		err = json.Unmarshal(valAsbytes, &keys)
		fmt.Println("Existing LC : %v", keys);
		if len(keys) == 0 {
			keysBytesToWrite, err := json.Marshal(&keys)
			err = stub.PutState("LCKeys", keysBytesToWrite)
			if err != nil {
				fmt.Println("Failed to initialize LC key collection")
			}
		} else {
			for _, key := range keys {
				valAsbytes, err := stub.GetState(key)
				if err == nil {
					var lc LC
					err = json.Unmarshal(valAsbytes, &lc)
					if err == nil {
						if lc.CurrentStatus == "" {
							lc.CurrentStatus = "Created"
							keysBytesToWrite, err := json.Marshal(&lc)
							if err == nil {
								err = stub.PutState(key, keysBytesToWrite)
								if err != nil {
									fmt.Println("Error writing LC to chain" + err.Error())
								}
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Println("Initialization complete")
	return nil, nil
}

func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

    // Handle different functions
    if function == "read" {                            //read a variable
        return t.read(stub, args)
    } else if function == "login" {
		return t.Login(stub, args)
    } else if function == "fileView" {
		return t.fileView(stub, args)	
    } else if function == "getAllLCs" {
		fmt.Println("Getting all LCs")
		allLCs, err := getAllLCs(stub)
		if err != nil {
			fmt.Println("Error from getAllLCs")
			return nil, err
		} else {
			allLCsBytes, err1 := json.Marshal(&allLCs)
			if err1 != nil {
				fmt.Println("Error marshalling allLCs")
				return nil, err1
			}
			fmt.Println("All success, returning allLCs")
			return allLCsBytes, nil
		}
    }
    fmt.Println("query did not find func: " + function)

    return nil, errors.New("Received unknown function query: " + function)
}

func (t *SimpleChaincode) Login(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Inside Login method")
	var username, password, jsonResp string
	var err error

    if len(args) < 2 {
        return nil, errors.New("Incorrect number of arguments. Expecting username and password")
    }
    username = args[0]
		password = args[1]

    valAsbytes, err := stub.GetState(username)
		if err != nil {
        jsonResp = "{\"Error\":\"Username not found " + username + "\"}"
        return nil, errors.New(jsonResp)
    }
		var existingUser User
		json.Unmarshal(valAsbytes, &existingUser)

		if existingUser.Password != password {
			jsonResp = "{\"Error\":\"Password does not match for " + username + "\"}"
			return nil, errors.New(jsonResp)
		}
		fmt.Println("Login complete")
    return valAsbytes, nil
}

func (t *SimpleChaincode) createLC(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Creating LC")

	if len(args) != 1 {
		fmt.Println("Error obtaining LC JSON string")
		return nil, errors.New("createLC accepts 1 argument (LCJSONString)")
	}
	
	var lc LC
	var err error
	fmt.Println("Unmarshalling LC");
	
	err = json.Unmarshal([]byte(args[0]), &lc)
	if err != nil {
		fmt.Println("error invalid LC string")
		return nil, errors.New("Invalid LC string")
	}
	
	lc.CurrentStatus = "Created"
	lc.ExporterBankApproved = false
	lc.ExporterDocsUploaded = false
	lc.CustomsApproved = false
	lc.PaymentComplete = false
	
	var shipmentId string
	shipmentId = lc.ShipmentId
	
	lcBytes, err := json.Marshal(&lc)
	if err != nil {
		fmt.Println("Error marshalling lc")
		return nil, errors.New("Error marshalling LC")
	}
	
	err = stub.PutState(lc.ShipmentId, lcBytes)
	if err != nil {
		fmt.Println("Error creating LC")
		return nil, errors.New("Error creating LC")
	}

	fmt.Println("Marshalling LC bytes to write")

	// Update the LC keys by adding the new key
	fmt.Println("Getting LC Keys")
	keysBytes, err := stub.GetState("LCKeys")
	if err != nil {
		fmt.Println("Error retrieving LC keys")
		return nil, errors.New("Error retrieving LC keys")
	}
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("Error unmarshel keys")
		return nil, errors.New("Error unmarshalling LC keys ")
	}

	fmt.Println("Appending the new key to LC Keys")
	foundKey := false
	for _, key := range keys {
		if key == lc.ShipmentId {
			foundKey = true
		}
	}
	if foundKey == false {
		keys = append(keys, lc.ShipmentId)
		keysBytesToWrite, err := json.Marshal(&keys)
		if err != nil {
			fmt.Println("Error marshalling keys")
			return nil, errors.New("Error marshalling the keys")
		}
		fmt.Println("Put state on PaperKeys")
		err = stub.PutState("LCKeys", keysBytesToWrite)
		if err != nil {
			fmt.Println("Error writting keys back")
			return nil, errors.New("Error writing the keys back")
		}
	}

	fmt.Println("LC Creation success", lc);
	
	var tosend = "LC created successfully for shipmentId :" + shipmentId + "." + stub.GetTxID();
    err = stub.SetEvent("invokeEvt", []byte(tosend))		
    if err != nil {
        return nil, err
    } else {
    	fmt.Println("Error nill event sent");
    }
	
	return nil, nil
}

func (t *SimpleChaincode) createUser(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Creating user")

	// Obtain the username to associate with the account
	if len(args) != 3 {
		fmt.Println("Error obtaining username/password/role")
		return nil, errors.New("createUser accepts 3 arguments (username, password, role)")
	}
	username := args[0]
	password := args[1]
	role := args[2]

	// Build an user object for the user
	var user = User{Username: username, Password: password, Role: role}
	userBytes, err := json.Marshal(&user)
	if err != nil {
		fmt.Println("error creating user" + user.Username)
		return nil, errors.New("Error creating user " + user.Username)
	}

	fmt.Println("Attempting to get state of any existing account for " + user.Username)
	existingBytes, err := stub.GetState(user.Username)
	if err == nil {
		var existingUser User
		err = json.Unmarshal(existingBytes, &existingUser)
		if err != nil {
			fmt.Println("Error unmarshalling user " + user.Username + "\n--->: " + err.Error())

			if strings.Contains(err.Error(), "unexpected end") {
				fmt.Println("No data means existing user found for " + user.Username + ", initializing user.")
				err = stub.PutState(user.Username, userBytes)

				if err == nil {
					fmt.Println("created user" + user.Username)
					return nil, nil
				} else {
					fmt.Println("failed to create initialize user for " + user.Username)
					return nil, errors.New("failed to initialize an account for " + user.Username + " => " + err.Error())
				}
			} else {
				return nil, errors.New("Error unmarshalling existing account " + user.Username)
			}
		} else {
			fmt.Println("Account already exists for " + user.Username + " " + existingUser.Username)
			return nil, errors.New("Can't reinitialize existing user " + user.Username)
		}
	} else {

		fmt.Println("No existing user found for " + user.Username + ", initializing user.")
		err = stub.PutState(user.Username, userBytes)

		if err == nil {
			fmt.Println("created user" + user.Username)
			return nil, nil
		} else {
			fmt.Println("failed to create initialize user for " + user.Username)
			return nil, errors.New("failed to initialize an user for " + user.Username + " => " + err.Error())
		}

	}
}

func (t *SimpleChaincode) uploadDocument(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, value, jsonResp string
	var err error
	fmt.Println("running uploadDocument()")

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3. name of the key and value to set")
	}

	key = args[0]+"_"+args[1]                            //rename for fun
	value = args[2]
	err = stub.PutState(key, []byte(value))  //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	
	//update document detail in LC JSON
	valAsbytes, err := stub.GetState(args[0])		
	if err != nil {
	    jsonResp = "{\"Error\":\"LC JSON not found " + args[1] + "\"}"
	    return nil, errors.New(jsonResp)
	}
	
	var existingLC LC
	err = json.Unmarshal(valAsbytes, &existingLC)
	if err == nil {
		existingLC.DocumentNames = append(existingLC.DocumentNames, args[1]);
	}
	if len(existingLC.DocumentNames) >= 2 {
		names := []string{args[0], "ExporterDocsUploaded", "true"}
		t.updateStatus(stub, names)
	}
	lcBytes, err := json.Marshal(&existingLC)
	if err != nil {
		fmt.Println("Error marshalling lc")
		return nil, errors.New("Error marshalling LC")
	}
	
	err = stub.PutState(existingLC.ShipmentId, lcBytes)
	if err != nil {
		fmt.Println("Error updating LC")
		return nil, errors.New("Error updating LC")
	}
	fmt.Println("Successfully updated")
	
	return nil, nil
}

func getAllLCs(stub shim.ChaincodeStubInterface) ([]LC, error) {

	var allLCs []LC

	// Get list of all the keys
	keysBytes, err := stub.GetState("LCKeys")
	if err != nil {
		fmt.Println("Error retrieving LC keys")
		return nil, errors.New("Error retrieving LC keys")
	}
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("Error unmarshalling LC keys" + err.Error())
		return nil, errors.New("Error unmarshalling LC keys")
	}

	// Get all the cps
	for _, value := range keys {
		lcBytes, err := stub.GetState(value)

		var lc LC
		err = json.Unmarshal(lcBytes, &lc)
		if err != nil {
			fmt.Println("Error retrieving lc " + value)
			return nil, errors.New("Error retrieving lc " + value)
		}

		fmt.Println("Appending LC" + value)
		allLCs = append(allLCs, lc)
	}

	return allLCs, nil
}

func (t *SimpleChaincode) updateStatus(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	var key, jsonResp string
	fmt.Println("running updateStatus")

	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3. updateStatus")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)		
	if err != nil {
	    jsonResp = "{\"Error\":\"File not found " + args[1] + "\"}"
	    return nil, errors.New(jsonResp)
	}
	
	var existingLC LC
	err = json.Unmarshal(valAsbytes, &existingLC)
	if err == nil {
		if args[1] == "ExporterBankApproved" {
			value, err := strconv.ParseBool(args[2])
			if err == nil {
				if value == true {
					existingLC.ExporterBankApproved = value
					existingLC.CurrentStatus = "ExporterBankApproved"
				} else {
					existingLC.ExporterBankApproved = value
					existingLC.CurrentStatus = "ExporterBankRejected"
				}
			}
		} else if args[1] == "ExporterDocsUploaded" {
			value, err := strconv.ParseBool(args[2])
			if err == nil {		
				existingLC.ExporterDocsUploaded = value
				existingLC.CurrentStatus = "ExporterDocsUploaded"
			}
		} else if args[1] == "CustomsApproved" {
			value, err := strconv.ParseBool(args[2])
			if err == nil {
				if value == true {
					existingLC.CustomsApproved = value
					existingLC.CurrentStatus = "CustomsApproved"
				} else {
					existingLC.CustomsApproved = value
					existingLC.CurrentStatus = "CustomsRejected"
				}
			}
		} else if args[1] == "PaymentComplete" {
			value, err := strconv.ParseBool(args[2])
			if err == nil {
				existingLC.PaymentComplete = value
				existingLC.CurrentStatus = "PaymentComplete"
			}
		}
	} else {
		return nil, errors.New("Error unmarshalling LC")
	}
	
	lcBytes, err := json.Marshal(&existingLC)
	if err != nil {
		fmt.Println("Error marshalling lc")
		return nil, errors.New("Error marshalling LC")
	}
	
	err = stub.PutState(existingLC.ShipmentId, lcBytes)
	if err != nil {
		fmt.Println("Error updating LC")
		return nil, errors.New("Error updating LC")
	}
	fmt.Println("Status successfully updated")
	return nil, nil
}

func (t *SimpleChaincode) fileView(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("Inside fileView method")
	var key, jsonResp string
	var err error

    if len(args) < 2 {
        return nil, errors.New("Incorrect number of arguments. Expecting shipmentId, fileName")
    }
	
	key = args[0]+"_"+args[1]

    valAsbytes, err := stub.GetState(key)
	if err != nil {
	    jsonResp = "{\"Error\":\"File not found " + args[1] + "\"}"
	    return nil, errors.New(jsonResp)
	}
	fmt.Println("FileLoading complete")
    return valAsbytes, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "write" {
		return t.write(stub, args)
	} else if function == "createUser" {
		return t.createUser(stub, args)
	} else if function == "createLC" {
		return t.createLC(stub, args)	
	} else if function == "uploadDocument" {
		fmt.Println("running uploadDocument>>>>>>>>>")
		return t.uploadDocument(stub, args)	
	} else if function == "updateStatus" {
		return t.updateStatus(stub, args)	
	}
	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation: " + function)
}

func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args[0]                            //rename for fun
	value = args[1]
	err = stub.PutState(key, []byte(value))  //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

    if len(args) != 1 {
        return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
    }

    key = args[0]
    valAsbytes, err := stub.GetState(key)
    if err != nil {
        jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
        return nil, errors.New(jsonResp)
    }

    return valAsbytes, nil
}
