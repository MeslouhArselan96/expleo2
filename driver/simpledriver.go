// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

// This package provides a simple example implementation of
// ProtocolDriver interface.
//
package driver

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"

	"github.com/edgexfoundry/device-sdk-go/v2/example/config"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
)

type ValidationSet struct {
	index                int32
	Cycle                int32
	Voltage_measured     float32
	Current_measured     float32
	Temperature_measured float32
	Voltage_load         float32
	Time                 float32
	Current_load         float32
	Capacity             float32
}

type SimpleDriver struct {
	lc            logger.LoggingClient
	asyncCh       chan<- *sdkModels.AsyncValues
	deviceCh      chan<- []sdkModels.DiscoveredDevice
	switchButton  bool
	xRotation     int32
	yRotation     int32
	zRotation     int32
	counter       interface{}
	stringArray   []string
	serviceConfig *config.ServiceConfig
	line_index    int32
	csvLines      [][]string
}

func getImageBytes(imgFile string, buf *bytes.Buffer) error {
	// Read existing image from file
	img, err := os.Open(imgFile)
	if err != nil {
		return err
	}
	defer img.Close()

	// TODO: Attach MediaType property, determine if decoding
	//  early is required (to optimize edge processing)

	// Expect "png" or "jpeg" image type
	imageData, imageType, err := image.Decode(img)
	if err != nil {
		return err
	}
	// Finished with file. Reset file pointer
	img.Seek(0, 0)
	if imageType == "jpeg" {
		err = jpeg.Encode(buf, imageData, nil)
		if err != nil {
			return err
		}
	} else if imageType == "png" {
		err = png.Encode(buf, imageData)
		if err != nil {
			return err
		}
	}
	return nil
}

// Initialize performs protocol-specific initialization for the device
// service.
func (s *SimpleDriver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModels.AsyncValues, deviceCh chan<- []sdkModels.DiscoveredDevice) error {
	s.lc = lc
	s.asyncCh = asyncCh
	s.deviceCh = deviceCh
	s.serviceConfig = &config.ServiceConfig{}
	s.counter = map[string]interface{}{
		"f1": "ABC",
		"f2": 123,
	}
	s.stringArray = []string{"foo", "bar"}
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	csvFile, err := os.Open("B0006.csv")
	if err != nil {
		return fmt.Errorf("unable to open csv file: %s", err.Error()) //////fichier chargé en memoire
	}
	defer csvFile.Close()

	s.line_index = 1
	s.csvLines, err = csv.NewReader(csvFile).ReadAll()
	if err != nil { ///////creation de l'index pour retrouver les lignes + lecture avec readAll

		return fmt.Errorf("unable to load csv information %s", err.Error())
	}
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	ds := service.RunningService()

	if err := ds.LoadCustomConfig(s.serviceConfig, "SimpleCustom"); err != nil {
		return fmt.Errorf("unable to load 'SimpleCustom' custom configuration: %s", err.Error())
	}

	lc.Infof("Custom config is: %v", s.serviceConfig.SimpleCustom)

	if err := s.serviceConfig.SimpleCustom.Validate(); err != nil {
		return fmt.Errorf("'SimpleCustom' custom configuration validation failed: %s", err.Error())
	}

	if err := ds.ListenForCustomConfigChanges(
		&s.serviceConfig.SimpleCustom.Writable,
		"SimpleCustom/Writable", s.ProcessCustomConfigChanges); err != nil {
		return fmt.Errorf("unable to listen for changes for 'SimpleCustom.Writable' custom configuration: %s", err.Error())
	}

	return nil
}

// ProcessCustomConfigChanges ...
func (s *SimpleDriver) ProcessCustomConfigChanges(rawWritableConfig interface{}) {
	updated, ok := rawWritableConfig.(*config.SimpleWritable)
	if !ok {
		s.lc.Error("unable to process custom config updates: Can not cast raw config to type 'SimpleWritable'")
		return
	}

	s.lc.Info("Received configuration updates for 'SimpleCustom.Writable' section")

	previous := s.serviceConfig.SimpleCustom.Writable
	s.serviceConfig.SimpleCustom.Writable = *updated

	if reflect.DeepEqual(previous, *updated) {
		s.lc.Info("No changes detected")
		return
	}

	// Now check to determine what changed.
	// In this example we only have the one writable setting,
	// so the check is not really need but left here as an example.
	// Since this setting is pulled from configuration each time it is need, no extra processing is required.
	// This may not be true for all settings, such as external host connection info, which
	// may require re-establishing the connection to the external host for example.
	if previous.DiscoverSleepDurationSecs != updated.DiscoverSleepDurationSecs {
		s.lc.Infof("DiscoverSleepDurationSecs changed to: %d", updated.DiscoverSleepDurationSecs)
	}
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (s *SimpleDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest) (res []*sdkModels.CommandValue, err error) {
	s.lc.Debugf("SimpleDriver.HandleReadCommands: protocols: %v resource: %v attributes: %v", protocols, reqs[0].DeviceResourceName, reqs[0].Attributes)

	if len(reqs) != 0 {
		res = make([]*sdkModels.CommandValue, 8)
		var cv *sdkModels.CommandValue
		var ranges [8]int
		out := s.csvLines[s.line_index]
		s.line_index += 1

		//index, err := strconv.ParseInt(out[0], 10, 64)
		Cycle, err := strconv.ParseInt(out[1], 10, 64)
		Voltage_measured, err := strconv.ParseFloat(out[2], 64)
		Current_measured, err := strconv.ParseFloat(out[3], 64)
		Temperature_measured, err := strconv.ParseFloat(out[4], 64)
		Voltage_load, err := strconv.ParseFloat(out[5], 64)
		Time, err := strconv.ParseFloat(out[6], 64)
		Current_load, err := strconv.ParseFloat(out[7], 64)
		Capacity, err := strconv.ParseFloat(out[8], 64)

		for i := range ranges {

			if err == nil {
				switch reqs[i].DeviceResourceName {
				case "Cycle":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeInt64, Cycle)
				case "Voltage_measured":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Voltage_measured)
				case "Current_measured":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Current_measured)
				case "Temperature_measured":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Temperature_measured)
				case "Voltage_load":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Voltage_load)
				case "Time":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Time)
				case "Current_load":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Current_load)
				case "Capacity":
					cv, _ = sdkModels.NewCommandValue(reqs[i].DeviceResourceName, common.ValueTypeFloat64, Capacity)
				}
			}

			res[i] = cv
		}
	}
	return
}

/* 	if len(reqs) == 1 {
	res = make([]*sdkModels.CommandValue, 1)
	if reqs[0].DeviceResourceName == "battery" {
		out := s.csvLines[s.line_index]
		s.line_index += 1
		index, err := strconv.ParseInt(out[0], 10, 32)
		Cycle, err := strconv.ParseInt(out[1], 10, 32)
		Voltage_measured, err := strconv.ParseFloat(out[2], 32)
		Current_measured, err := strconv.ParseFloat(out[3], 32)
		Temperature_measured, err := strconv.ParseFloat(out[4], 32)
		Voltage_load, err := strconv.ParseFloat(out[5], 32)
		Time, err := strconv.ParseFloat(out[6], 32)
		Current_load, err := strconv.ParseFloat(out[7], 32)
		Capacity, err := strconv.ParseFloat(out[8], 32)

		line := ValidationSet{
			index:                int32(index),
			Cycle:                int32(Cycle),
			Voltage_measured:     float32(Voltage_measured),
			Current_measured:     float32(Current_measured),
			Temperature_measured: float32(Temperature_measured),
			Voltage_load:         float32(Voltage_load),
			Time:                 float32(Time),
			Current_load:         float32(Current_load),
			Capacity:             float32(Capacity),
		}

		//json_object, err := json.Marshal(line)

		if err == nil {
			cv, _ := sdkModels.NewCommandValue(reqs[0].DeviceResourceName, common.ValueTypeObject, line)
			res[0] = cv
		}

	}

} */
/*if len(reqs) == 1 {
	res = make([]*sdkModels.CommandValue, 1)
	if reqs[0].DeviceResourceName == "Cycle" {
		out := "10"
		buffer, err := strconv.ParseInt(out, 10, 64)
		buffer = int64(buffer)
		if err == nil {
			cv, _ := sdkModels.NewCommandValue(reqs[0].DeviceResourceName, common.ValueTypeInt64, buffer)
			res[0] = cv
		}
	} else if reqs[0].DeviceResourceName == "Capacity" {

		out := "20"
		buffer, err := strconv.ParseInt(out, 10, 64)
		buffer = int64(buffer)

		if err == nil {
			cv, _ := sdkModels.NewCommandValue(reqs[0].DeviceResourceName, common.ValueTypeInt64, buffer)
			res[0] = cv
		}
	}
}*/

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource.
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (s *SimpleDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest,
	params []*sdkModels.CommandValue) error {
	var err error

	for i, r := range reqs {
		s.lc.Debugf("SimpleDriver.HandleWriteCommands: protocols: %v, resource: %v, parameters: %v, attributes: %v", protocols, reqs[i].DeviceResourceName, params[i], reqs[i].Attributes)
		switch r.DeviceResourceName {
		case "SwitchButton":
			if s.switchButton, err = params[i].BoolValue(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be Boolean, parameter: %s", params[0].String())
				return err
			}
		case "Xrotation":
			if s.xRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "Yrotation":
			if s.yRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "Zrotation":
			if s.zRotation, err = params[i].Int32Value(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be Int32, parameter: %s", params[i].String())
				return err
			}
		case "StringArray":
			if s.stringArray, err = params[i].StringArrayValue(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be string array, parameter: %s", params[i].String())
				return err
			}
		case "Uint8Array":
			v, err := params[i].Uint8ArrayValue()
			if err == nil {
				s.lc.Debugf("Uint8 array value from write command: ", v)
			} else {
				return err
			}
		case "Counter":
			if s.counter, err = params[i].ObjectValue(); err != nil {
				err := fmt.Errorf("SimpleDriver.HandleWriteCommands; the data type of parameter should be Object, parameter: %s", params[i].String())
				return err
			}
		}
	}

	return nil
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (s *SimpleDriver) Stop(force bool) error {
	// Then Logging Client might not be initialized
	if s.lc != nil {
		s.lc.Debugf("SimpleDriver.Stop called: force=%v", force)
	}
	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (s *SimpleDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	s.lc.Debugf("a new Device is added: %s", deviceName)
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (s *SimpleDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	s.lc.Debugf("Device %s is updated", deviceName)
	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (s *SimpleDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	s.lc.Debugf("Device %s is removed", deviceName)
	return nil
}

// Discover triggers protocol specific device discovery, which is an asynchronous operation.
// Devices found as part of this discovery operation are written to the channel devices.
func (s *SimpleDriver) Discover() {
	proto := make(map[string]models.ProtocolProperties)
	proto["other"] = map[string]string{"Address": "simple02", "Port": "301"}

	device2 := sdkModels.DiscoveredDevice{
		Name:        "Simple-Device02",
		Protocols:   proto,
		Description: "found by discovery",
		Labels:      []string{"auto-discovery"},
	}

	proto = make(map[string]models.ProtocolProperties)
	proto["other"] = map[string]string{"Address": "simple03", "Port": "399"}

	device3 := sdkModels.DiscoveredDevice{
		Name:        "Simple-Device03",
		Protocols:   proto,
		Description: "found by discovery",
		Labels:      []string{"auto-discovery"},
	}

	res := []sdkModels.DiscoveredDevice{device2, device3}

	time.Sleep(time.Duration(s.serviceConfig.SimpleCustom.Writable.DiscoverSleepDurationSecs) * time.Second)
	s.deviceCh <- res
}
