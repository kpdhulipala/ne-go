package ne

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/equinix/ne-go/internal/api"
	"github.com/equinix/rest-go"
)

const (
	//DeviceManagementTypeSelf indicates device management mode where customer
	//fully manages the device
	DeviceManagementTypeSelf = "SELF-CONFIGURED"
	//DeviceManagementTypeEquinix indicates device management mode where device
	//connectivity and services are managed by Equinix
	DeviceManagementTypeEquinix = "EQUINIX-CONFIGURED"
	//DeviceLicenseModeSubscription indicates device software license mode where
	//Equinix provides software license in a form of subscription
	DeviceLicenseModeSubscription = "Sub"
	//DeviceLicenseModeBYOL indicates device software license mode where
	//customer provides his own, externally procured device license
	DeviceLicenseModeBYOL = "BYOL"
)

type restDeviceUpdateRequest struct {
	uuid                string
	deviceFields        map[string]interface{}
	additionalBandwidth *int
	aclTemplateID       *string
	mgmtAclTemplateUuid *string
	c                   RestClient
}

// CreateDevice creates given Network Edge device and returns its UUID upon successful creation
func (c RestClient) CreateDevice(device Device) (*string, error) {
	path := "/ne/v1/devices"
	reqBody := createDeviceRequest(device)
	respBody := api.DeviceRequestResponse{}
	req := c.R().SetBody(&reqBody).SetResult(&respBody)
	if err := c.Execute(req, http.MethodPost, path); err != nil {
		return nil, err
	}
	return respBody.UUID, nil
}

// CreateRedundantDevice creates HA device setup from given primary and secondary devices and
// returns their UUIDS upon successful creation
func (c RestClient) CreateRedundantDevice(primary Device, secondary Device) (*string, *string, error) {
	path := "/ne/v1/devices"
	reqBody := createRedundantDeviceRequest(primary, secondary)
	respBody := api.DeviceRequestResponse{}
	req := c.R().SetBody(&reqBody).SetResult(&respBody)
	if err := c.Execute(req, http.MethodPost, path); err != nil {
		return nil, nil, err
	}
	return respBody.UUID, respBody.SecondaryUUID, nil
}

func (c RestClient) AddSecondary(primaryUuid string, secondary Device) (*string, error) {
	updateErr := UpdateError{}
	secondaryUuid, err := c.addSecondaryDevice(primaryUuid, secondary)
	if err != nil {
		updateErr.AddChangeError(changeTypeUpdate, "secondary", secondary, err)
	}
	return secondaryUuid, nil
}

// GetDevice fetches details of a device with a given UUID
func (c RestClient) GetDevice(uuid string) (*Device, error) {
	path := "/ne/v1/devices/" + url.PathEscape(uuid)
	result := api.Device{}
	request := c.R().SetResult(&result)
	if err := c.Execute(request, http.MethodGet, path); err != nil {
		return nil, err
	}
	return mapDeviceAPIToDomain(result), nil
}

// GetDevices retrieves list of devices (along with their details) with given list of statuses
func (c RestClient) GetDevices(statuses []string) ([]Device, error) {
	path := "/ne/v1/devices"
	content, err := c.GetOffsetPaginated(path, &api.DevicesResponse{},
		rest.DefaultOffsetPagingConfig().
			SetAdditionalParams(map[string]string{"status": buildQueryParamValueString(statuses)}))
	if err != nil {
		return nil, err
	}
	transformed := make([]Device, len(content))
	for i := range content {
		transformed[i] = *mapDeviceAPIToDomain(content[i].(api.Device))
	}
	return transformed, nil
}

// GetDeviceAdditionalBandwidthDetails retrives details of given device's additional bandwidth
func (c RestClient) GetDeviceAdditionalBandwidthDetails(uuid string) (*DeviceAdditionalBandwidthDetails, error) {
	path := fmt.Sprintf("/ne/v1/devices/%s/additionalBandwidths", url.PathEscape(uuid))
	result := api.DeviceAdditionalBandwidthResponse{}
	request := c.R().SetResult(&result)
	if err := c.Execute(request, http.MethodGet, path); err != nil {
		return nil, err
	}
	return mapDeviceAdditionalBandwidthAPIToDomain(result), nil
}

// GetDeviceACLDetails retrives device acl template provisioning status
func (c RestClient) GetDeviceACLDetails(uuid string) (*DeviceACLDetails, error) {
	path := fmt.Sprintf("/ne/v1/devices/%s/acl", url.PathEscape(uuid))
	result := api.DeviceACLResponse{}
	request := c.R().SetResult(&result)
	if err := c.Execute(request, http.MethodGet, path); err != nil {
		return nil, err
	}
	return mapDeviceACLAPIToDomain(result), nil
}

// NewDeviceUpdateRequest creates new composite update request for a device with a given UUID
func (c RestClient) NewDeviceUpdateRequest(uuid string) DeviceUpdateRequest {
	return &restDeviceUpdateRequest{
		uuid:         uuid,
		deviceFields: make(map[string]interface{}),
		c:            c}
}

// DeleteDevice deletes device with a given UUID
func (c RestClient) DeleteDevice(uuid string) error {
	path := "/ne/v1/devices/" + url.PathEscape(uuid)
	req := c.R().SetQueryParam("deleteRedundantDevice", "true")
	if err := c.Execute(req, http.MethodDelete, path); err != nil {
		return err
	}
	return nil
}

func (c RestClient) DeleteSecondaryDevice(uuid string) error {
	path := "/ne/v1/devices/" + url.PathEscape(uuid)
	req := c.R().SetQueryParam("deleteRedundantDevice", "false")
	if err := c.Execute(req, http.MethodDelete, path); err != nil {
		return err
	}
	return nil
}

// WithDeviceName sets new device name in a composite device update request
func (req *restDeviceUpdateRequest) WithDeviceName(deviceName string) DeviceUpdateRequest {
	req.deviceFields["deviceName"] = deviceName
	return req
}

// WithTermLength sets new term length in a composite device update request
func (req *restDeviceUpdateRequest) WithTermLength(termLength int) DeviceUpdateRequest {
	req.deviceFields["termLength"] = termLength
	return req
}

// WithNotifications sets new notifications in a composite device update request
func (req *restDeviceUpdateRequest) WithNotifications(notifications []string) DeviceUpdateRequest {
	req.deviceFields["notifications"] = notifications
	return req
}

// WithCore sets new core count in a composite device update request
func (req *restDeviceUpdateRequest) WithCore(core int) DeviceUpdateRequest {
	req.deviceFields["core"] = core
	return req
}

// WithClusterName sets new cluster name in a composite device update request
func (req *restDeviceUpdateRequest) WithClusterName(clusterName string) DeviceUpdateRequest {
	req.deviceFields["clusterName"] = clusterName
	return req
}

// WithAdditionalBandwidth sets new additional bandwidth in a composite device update request
func (req *restDeviceUpdateRequest) WithAdditionalBandwidth(additionalBandwidth int) DeviceUpdateRequest {
	req.additionalBandwidth = &additionalBandwidth
	return req
}

// WithACLTemplate sets new ACL template identifier in a composite device update request
func (req *restDeviceUpdateRequest) WithACLTemplate(templateID string) DeviceUpdateRequest {
	req.aclTemplateID = &templateID
	return req
}

// WithMgmtAclTemplate sets new MGMT ACL template identifier in a composite device update request
func (req *restDeviceUpdateRequest) WithMgmtAclTemplate(mgmtAclTemplateUuid string) DeviceUpdateRequest {
	req.mgmtAclTemplateUuid = &mgmtAclTemplateUuid
	return req
}

// Execute attempts to update device according new data set in composite update request.
// This is not atomic operation and if any update will fail, other changes won't be reverted.
// UpdateError will be returned if any of requested data failed to update
func (req *restDeviceUpdateRequest) Execute() error {
	updateErr := UpdateError{}
	if err := req.c.replaceDeviceFields(req.uuid, req.deviceFields); err != nil {
		updateErr.AddChangeError(changeTypeUpdate, "deviceFields", req.deviceFields, err)
	}
	if req.aclTemplateID != nil || req.mgmtAclTemplateUuid != nil {
		if err := req.c.replaceDeviceACLTemplate(req.uuid, req.aclTemplateID, req.mgmtAclTemplateUuid); err != nil {
			if req.aclTemplateID != nil {
				updateErr.AddChangeError(changeTypeUpdate, "aclTemplateUuid", *req.aclTemplateID, err)
			}
			if req.mgmtAclTemplateUuid != nil {
				updateErr.AddChangeError(changeTypeUpdate, "mgmtAclTemplateUuid", *req.mgmtAclTemplateUuid, err)
			}
		}
	}
	if req.additionalBandwidth != nil {
		if err := req.c.replaceDeviceAdditionalBandwidth(req.uuid, *req.additionalBandwidth); err != nil {
			updateErr.AddChangeError(changeTypeUpdate, "additionalBandwidth", req.additionalBandwidth, err)
		}
	}
	if updateErr.ChangeErrorsCount() > 0 {
		return updateErr
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported package methods
//_______________________________________________________________________

func mapDeviceAPIToDomain(apiDevice api.Device) *Device {
	device := Device{}
	device.UUID = apiDevice.UUID
	device.Name = apiDevice.Name
	device.TypeCode = apiDevice.DeviceTypeCode
	device.Status = apiDevice.Status
	device.LicenseStatus = apiDevice.LicenseStatus
	device.MetroCode = apiDevice.MetroCode
	device.IBX = apiDevice.IBX
	device.Region = apiDevice.Region
	device.Throughput = apiDevice.Throughput
	device.ThroughputUnit = apiDevice.ThroughputUnit
	device.HostName = apiDevice.HostName
	device.PackageCode = apiDevice.PackageCode
	device.Version = apiDevice.Version
	if apiDevice.LicenseType != nil {
		if *apiDevice.LicenseType == DeviceLicenseModeBYOL {
			device.IsBYOL = Bool(true)
		} else {
			device.IsBYOL = Bool(false)
		}
	}
	device.LicenseToken = apiDevice.LicenseToken
	device.LicenseFileID = apiDevice.LicenseFileID
	device.ACLTemplateUUID = apiDevice.ACLTemplateUUID
	device.MgmtAclTemplateUuid = apiDevice.MgmtAclTemplateUUID
	device.SSHIPAddress = apiDevice.SSHIPAddress
	device.SSHIPFqdn = apiDevice.SSHIPFqdn
	device.AccountNumber = apiDevice.AccountNumber
	device.Notifications = apiDevice.Notifications
	device.PurchaseOrderNumber = apiDevice.PurchaseOrderNumber
	device.RedundancyType = apiDevice.RedundancyType
	device.RedundantUUID = apiDevice.RedundantUUID
	device.TermLength = apiDevice.TermLength
	device.ProjectID = apiDevice.ProjectID
	device.AdditionalBandwidth = apiDevice.AdditionalBandwidth
	device.WanInterfaceId = apiDevice.SshInterfaceID
	device.OrderReference = apiDevice.OrderReference
	device.InterfaceCount = apiDevice.InterfaceCount
	if apiDevice.Core != nil {
		device.CoreCount = apiDevice.Core.Core
	}
	if apiDevice.Core != nil {
		device.Tier = apiDevice.Core.Tier
	}
	if apiDevice.DeviceManagementType != nil {
		if *apiDevice.DeviceManagementType == DeviceManagementTypeSelf {
			device.IsSelfManaged = Bool(true)
		} else {
			device.IsSelfManaged = Bool(false)
		}
	}
	device.Connectivity = apiDevice.Connectivity
	device.Interfaces = mapDeviceInterfacesAPIToDomain(apiDevice.Interfaces)
	device.VendorConfiguration = apiDevice.VendorConfig
	device.UserPublicKey = mapDeviceUserPublicKeyAPIToDomain(apiDevice.UserPublicKey)
	device.ASN = apiDevice.ASN
	device.ZoneCode = apiDevice.ZoneCode
	device.ClusterDetails = mapDeviceClusterDetailsAPIToDomain(apiDevice.ClusterDetails)
	device.DiverseFromDeviceUUID = apiDevice.DiverseFromDeviceUUID
	device.DiverseFromDeviceName = apiDevice.DiverseFromDeviceName
	device.IsGenerateDefaultPassword = apiDevice.IsGenerateDefaultPassword
	return &device
}

func mapDeviceInterfacesAPIToDomain(apiInterfaces []api.DeviceInterface) []DeviceInterface {
	transformed := make([]DeviceInterface, len(apiInterfaces))
	for i := range apiInterfaces {
		transformed[i] = DeviceInterface{
			ID:                apiInterfaces[i].ID,
			Name:              apiInterfaces[i].Name,
			Status:            apiInterfaces[i].Status,
			OperationalStatus: apiInterfaces[i].OperationalStatus,
			MACAddress:        apiInterfaces[i].MACAddress,
			IPAddress:         apiInterfaces[i].IPAddress,
			AssignedType:      apiInterfaces[i].AssignedType,
			Type:              apiInterfaces[i].Type,
		}
	}
	return transformed
}

func mapDeviceUserPublicKeyAPIToDomain(apiUserKey *api.DeviceUserPublicKey) *DeviceUserPublicKey {
	if apiUserKey == nil {
		return nil
	}
	return &DeviceUserPublicKey{
		Username: apiUserKey.Username,
		KeyName:  apiUserKey.KeyName,
	}
}

func mapDeviceUserPublicKeyDomainToAPI(userKey *DeviceUserPublicKey) *api.DeviceUserPublicKeyRequest {
	if userKey == nil {
		return nil
	}
	return &api.DeviceUserPublicKeyRequest{
		Username: userKey.Username,
		KeyName:  userKey.KeyName,
	}
}

func mapDeviceClusterDetailsAPIToDomain(apiClusterDetails *api.ClusterDetails) *ClusterDetails {
	if apiClusterDetails == nil {
		return nil
	}
	clusterDetails := ClusterDetails{}
	clusterDetails.ClusterId = apiClusterDetails.ClusterID
	clusterDetails.ClusterName = apiClusterDetails.ClusterName
	clusterDetails.NumOfNodes = apiClusterDetails.NumOfNodes
	apiNodes := apiClusterDetails.Nodes
	transformed := make([]ClusterNode, len(apiNodes))
	clusterNodeDetails := make([]*ClusterNodeDetail, len(apiNodes))
	for i := range apiNodes {
		transformed[i] = ClusterNode{
			UUID:                apiNodes[i].UUID,
			Name:                apiNodes[i].Name,
			Node:                apiNodes[i].Node,
			AdminPassword:       apiNodes[i].AdminPassword,
			VendorConfiguration: apiNodes[i].VendorConfiguration,
		}
		clusterNodeDetails[i] = &ClusterNodeDetail{
			UUID:                apiNodes[i].UUID,
			Name:                apiNodes[i].Name,
			VendorConfiguration: apiNodes[i].VendorConfiguration,
		}
	}
	clusterDetails.Nodes = transformed
	clusterDetails.Node0 = clusterNodeDetails[0]
	clusterDetails.Node1 = clusterNodeDetails[1]
	return &clusterDetails
}

func mapDeviceClusterDetailsDomainToAPI(clusterDetails *ClusterDetails) *api.ClusterDetailsRequest {
	if clusterDetails == nil {
		return nil
	}
	req := api.ClusterDetailsRequest{}
	req.ClusterName = clusterDetails.ClusterName
	clusterNodeDetailsRequest := make(map[string]api.ClusterNodeDetailRequest)
	clusterNodeDetailsRequest["node0"] = *mapDeviceClusterNodeDetailDomainToAPI(clusterDetails.Node0)
	clusterNodeDetailsRequest["node1"] = *mapDeviceClusterNodeDetailDomainToAPI(clusterDetails.Node1)
	req.ClusterNodeDetails = clusterNodeDetailsRequest
	return &req
}

func mapDeviceClusterNodeDetailDomainToAPI(clusterNodeDetail *ClusterNodeDetail) *api.ClusterNodeDetailRequest {
	if clusterNodeDetail == nil {
		return nil
	}
	return &api.ClusterNodeDetailRequest{
		VendorConfiguration: clusterNodeDetail.VendorConfiguration,
		LicenseFileID:       clusterNodeDetail.LicenseFileId,
		LicenseToken:        clusterNodeDetail.LicenseToken,
	}
}

func createDeviceRequest(device Device) api.DeviceRequest {
	req := api.DeviceRequest{}
	req.Throughput = device.Throughput
	req.Tier = device.Tier
	req.ThroughputUnit = device.ThroughputUnit
	req.MetroCode = device.MetroCode
	req.DeviceTypeCode = device.TypeCode
	if device.TermLength != nil {
		termLengthString := strconv.Itoa(*device.TermLength)
		req.TermLength = &termLengthString
	}
	if device.IsBYOL != nil {
		if *device.IsBYOL {
			req.LicenseMode = String(DeviceLicenseModeBYOL)
		} else {
			req.LicenseMode = String(DeviceLicenseModeSubscription)
		}
	}
	req.LicenseToken = device.LicenseToken
	req.LicenseFileID = device.LicenseFileID
	req.CloudInitFileID = device.CloudInitFileID
	req.PackageCode = device.PackageCode
	req.VirtualDeviceName = device.Name
	req.Notifications = device.Notifications
	req.HostNamePrefix = device.HostName
	req.OrderReference = device.OrderReference
	req.PurchaseOrderNumber = device.PurchaseOrderNumber
	req.AccountNumber = device.AccountNumber
	req.Version = device.Version
	req.InterfaceCount = device.InterfaceCount
	if device.IsSelfManaged != nil {
		if *device.IsSelfManaged {
			req.DeviceManagementType = String(DeviceManagementTypeSelf)
		} else {
			req.DeviceManagementType = String(DeviceManagementTypeEquinix)
		}
	}
	req.Connectivity = device.Connectivity
	req.ProjectID = device.ProjectID
	req.Core = device.CoreCount
	req.AdditionalBandwidth = device.AdditionalBandwidth
	req.SshInterfaceId = device.WanInterfaceId
	req.ACLTemplateUUID = device.ACLTemplateUUID
	req.MgmtAclTemplateUUID = device.MgmtAclTemplateUuid
	req.VendorConfig = device.VendorConfiguration
	req.UserPublicKey = mapDeviceUserPublicKeyDomainToAPI(device.UserPublicKey)
	req.ClusterDetails = mapDeviceClusterDetailsDomainToAPI(device.ClusterDetails)
	req.DiverseFromDeviceUUID = device.DiverseFromDeviceUUID
	req.IsGenerateDefaultPassword = device.IsGenerateDefaultPassword
	return req
}

func createRedundantDeviceRequest(primary Device, secondary Device) api.DeviceRequest {
	req := createDeviceRequest(primary)
	secReq := api.SecondaryDeviceRequest{}
	secReq.MetroCode = secondary.MetroCode
	secReq.LicenseToken = secondary.LicenseToken
	secReq.LicenseFileID = secondary.LicenseFileID
	secReq.CloudInitFileID = secondary.CloudInitFileID
	secReq.VirtualDeviceName = secondary.Name
	secReq.Notifications = secondary.Notifications
	secReq.HostNamePrefix = secondary.HostName
	secReq.AccountNumber = secondary.AccountNumber
	secReq.AdditionalBandwidth = secondary.AdditionalBandwidth
	secReq.SshInterfaceID = secondary.WanInterfaceId
	if secReq.SshInterfaceID == nil {
		secReq.SshInterfaceID = req.SshInterfaceId
	}
	secReq.ACLTemplateUUID = secondary.ACLTemplateUUID
	secReq.MgmtAclTemplateUUID = secondary.MgmtAclTemplateUuid
	secReq.VendorConfig = secondary.VendorConfiguration
	secReq.UserPublicKey = mapDeviceUserPublicKeyDomainToAPI(secondary.UserPublicKey)
	req.Secondary = &secReq
	return req
}

func addSecondaryDeviceRequest(primaryDeviceUuid string, secondary Device) api.AddSecondaryRequest {
	req := api.AddSecondaryRequest{}
	req.PrimaryDeviceUUID = &primaryDeviceUuid
	secReq := api.SecondaryDeviceRequest{}
	secReq.MetroCode = secondary.MetroCode
	secReq.LicenseToken = secondary.LicenseToken
	secReq.LicenseFileID = secondary.LicenseFileID
	secReq.CloudInitFileID = secondary.CloudInitFileID
	secReq.VirtualDeviceName = secondary.Name
	secReq.Notifications = secondary.Notifications
	secReq.HostNamePrefix = secondary.HostName
	secReq.AccountNumber = secondary.AccountNumber
	secReq.AdditionalBandwidth = secondary.AdditionalBandwidth
	secReq.SshInterfaceID = secondary.WanInterfaceId
	secReq.ACLTemplateUUID = secondary.ACLTemplateUUID
	secReq.MgmtAclTemplateUUID = secondary.MgmtAclTemplateUuid
	secReq.VendorConfig = secondary.VendorConfiguration
	secReq.UserPublicKey = mapDeviceUserPublicKeyDomainToAPI(secondary.UserPublicKey)
	req.Secondary = &secReq
	return req
}

func (c RestClient) replaceDeviceACLTemplate(uuid string, wanAclTemplateUuid *string, mgmtAclTemplateUuid *string) error {
	path := "/ne/v1/devices/" + url.PathEscape(uuid) + "/acl"
	reqBody := api.DeviceACLTemplateRequest{
		TemplateUUID:        wanAclTemplateUuid,
		MgmtAclTemplateUUID: mgmtAclTemplateUuid,
	}
	req := c.R().SetBody(reqBody)
	if err := c.Execute(req, http.MethodPatch, path); err != nil {
		return err
	}
	return nil
}

func (c RestClient) replaceDeviceAdditionalBandwidth(uuid string, bandwidth int) error {
	path := fmt.Sprintf("/ne/v1/devices/%s/additionalBandwidths", url.PathEscape(uuid))
	reqBody := api.DeviceAdditionalBandwidthUpdateRequest{AdditionalBandwidth: &bandwidth}
	req := c.R().SetBody(reqBody)
	if err := c.Execute(req, http.MethodPut, path); err != nil {
		return err
	}
	return nil
}

func (c RestClient) replaceDeviceFields(uuid string, fields map[string]interface{}) error {
	reqBody := api.DeviceUpdateRequest{}
	okToSend := false
	if v, ok := fields["deviceName"]; ok {
		reqBody.VirtualDeviceName = String(v.(string))
		okToSend = true
	}
	if v, ok := fields["termLength"]; ok {
		reqBody.TermLength = Int(v.(int))
		okToSend = true
	}
	if v, ok := fields["notifications"]; ok {
		reqBody.Notifications = v.([]string)
		okToSend = true
	}
	if v, ok := fields["core"]; ok {
		reqBody.Core = Int(v.(int))
		okToSend = true
	}
	if v, ok := fields["clusterName"]; ok {
		reqBody.ClusterName = String(v.(string))
		okToSend = true
	}
	if okToSend {
		path := "/ne/v1/devices/" + uuid
		req := c.R().SetBody(&reqBody)
		if err := c.Execute(req, http.MethodPatch, path); err != nil {
			return err
		}
	}
	return nil
}

func (c RestClient) addSecondaryDevice(uuid string, secondary Device) (*string, error) {
	//build logic for post secondary device
	//uuid = primary device uuid
	path := "/ne/v1/devices"
	reqBody := addSecondaryDeviceRequest(uuid, secondary)
	respBody := api.DeviceRequestResponse{}
	req := c.R().SetBody(&reqBody).SetResult(&respBody)
	if err := c.Execute(req, http.MethodPost, path); err != nil {
		return nil, err
	}
	return respBody.SecondaryUUID, nil
}

func buildQueryParamValueString(values []string) string {
	var sb strings.Builder
	for i := range values {
		sb.WriteString(url.QueryEscape(values[i]))
		if i < len(values)-1 {
			sb.WriteString(",")
		}
	}
	return sb.String()
}

func mapDeviceAdditionalBandwidthAPIToDomain(apiDetails api.DeviceAdditionalBandwidthResponse) *DeviceAdditionalBandwidthDetails {
	return &DeviceAdditionalBandwidthDetails{
		AdditionalBandwidth: apiDetails.AdditionalBandwidth,
		Status:              apiDetails.Status,
	}
}

func mapDeviceACLAPIToDomain(apiDetails api.DeviceACLResponse) *DeviceACLDetails {
	return &DeviceACLDetails{
		Status: apiDetails.Status,
	}
}
