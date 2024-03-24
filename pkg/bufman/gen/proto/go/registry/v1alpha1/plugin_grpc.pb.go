// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// registry/v1alpha1/plugin.proto is a deprecated file.

package registryv1alpha1

import (
	context "context"
)

import (
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	PluginService_ListPlugins_FullMethodName                = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListPlugins"
	PluginService_ListUserPlugins_FullMethodName            = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListUserPlugins"
	PluginService_ListOrganizationPlugins_FullMethodName    = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListOrganizationPlugins"
	PluginService_GetPluginVersion_FullMethodName           = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/GetPluginVersion"
	PluginService_ListPluginVersions_FullMethodName         = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListPluginVersions"
	PluginService_GetPlugin_FullMethodName                  = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/GetPlugin"
	PluginService_DeletePlugin_FullMethodName               = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/DeletePlugin"
	PluginService_GetTemplate_FullMethodName                = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/GetTemplate"
	PluginService_ListTemplates_FullMethodName              = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListTemplates"
	PluginService_ListTemplatesUserCanAccess_FullMethodName = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListTemplatesUserCanAccess"
	PluginService_ListUserTemplates_FullMethodName          = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListUserTemplates"
	PluginService_ListOrganizationTemplates_FullMethodName  = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListOrganizationTemplates"
	PluginService_GetTemplateVersion_FullMethodName         = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/GetTemplateVersion"
	PluginService_ListTemplateVersions_FullMethodName       = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/ListTemplateVersions"
	PluginService_DeleteTemplate_FullMethodName             = "/bufman.dubbo.apache.org.registry.v1alpha1.PluginService/DeleteTemplate"
)

// PluginServiceClient is the client API for PluginService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PluginServiceClient interface {
	// ListPlugins returns all the plugins available to the user. This includes
	// public plugins, those uploaded to organizations the user is part of,
	// and any plugins uploaded directly by the user.
	ListPlugins(ctx context.Context, in *ListPluginsRequest, opts ...grpc.CallOption) (*ListPluginsResponse, error)
	// ListUserPlugins lists all plugins belonging to a user.
	ListUserPlugins(ctx context.Context, in *ListUserPluginsRequest, opts ...grpc.CallOption) (*ListUserPluginsResponse, error)
	// ListOrganizationPlugins lists all plugins for an organization.
	ListOrganizationPlugins(ctx context.Context, in *ListOrganizationPluginsRequest, opts ...grpc.CallOption) (*ListOrganizationPluginsResponse, error)
	// GetPluginVersion returns the plugin version, if found.
	GetPluginVersion(ctx context.Context, in *GetPluginVersionRequest, opts ...grpc.CallOption) (*GetPluginVersionResponse, error)
	// ListPluginVersions lists all the versions available for the specified plugin.
	ListPluginVersions(ctx context.Context, in *ListPluginVersionsRequest, opts ...grpc.CallOption) (*ListPluginVersionsResponse, error)
	// GetPlugin returns the plugin, if found.
	GetPlugin(ctx context.Context, in *GetPluginRequest, opts ...grpc.CallOption) (*GetPluginResponse, error)
	// DeletePlugin deletes the plugin, if it exists. Note that deleting
	// a plugin may cause breaking changes for templates using that plugin,
	// and should be done with extreme care.
	DeletePlugin(ctx context.Context, in *DeletePluginRequest, opts ...grpc.CallOption) (*DeletePluginResponse, error)
	// GetTemplate returns the template, if found.
	GetTemplate(ctx context.Context, in *GetTemplateRequest, opts ...grpc.CallOption) (*GetTemplateResponse, error)
	// ListTemplates returns all the templates available to the user. This includes
	// public templates, those owned by organizations the user is part of,
	// and any created directly by the user.
	ListTemplates(ctx context.Context, in *ListTemplatesRequest, opts ...grpc.CallOption) (*ListTemplatesResponse, error)
	// ListTemplatesUserCanAccess is like ListTemplates, but does not return
	// public templates.
	ListTemplatesUserCanAccess(ctx context.Context, in *ListTemplatesUserCanAccessRequest, opts ...grpc.CallOption) (*ListTemplatesUserCanAccessResponse, error)
	// ListUserPlugins lists all templates belonging to a user.
	ListUserTemplates(ctx context.Context, in *ListUserTemplatesRequest, opts ...grpc.CallOption) (*ListUserTemplatesResponse, error)
	// ListOrganizationTemplates lists all templates for an organization.
	ListOrganizationTemplates(ctx context.Context, in *ListOrganizationTemplatesRequest, opts ...grpc.CallOption) (*ListOrganizationTemplatesResponse, error)
	// GetTemplateVersion returns the template version, if found.
	GetTemplateVersion(ctx context.Context, in *GetTemplateVersionRequest, opts ...grpc.CallOption) (*GetTemplateVersionResponse, error)
	// ListTemplateVersions lists all the template versions available for the specified template.
	ListTemplateVersions(ctx context.Context, in *ListTemplateVersionsRequest, opts ...grpc.CallOption) (*ListTemplateVersionsResponse, error)
	// DeleteTemplate deletes the template, if it exists.
	DeleteTemplate(ctx context.Context, in *DeleteTemplateRequest, opts ...grpc.CallOption) (*DeleteTemplateResponse, error)
}

type pluginServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPluginServiceClient(cc grpc.ClientConnInterface) PluginServiceClient {
	return &pluginServiceClient{cc}
}

func (c *pluginServiceClient) ListPlugins(ctx context.Context, in *ListPluginsRequest, opts ...grpc.CallOption) (*ListPluginsResponse, error) {
	out := new(ListPluginsResponse)
	err := c.cc.Invoke(ctx, PluginService_ListPlugins_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListUserPlugins(ctx context.Context, in *ListUserPluginsRequest, opts ...grpc.CallOption) (*ListUserPluginsResponse, error) {
	out := new(ListUserPluginsResponse)
	err := c.cc.Invoke(ctx, PluginService_ListUserPlugins_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListOrganizationPlugins(ctx context.Context, in *ListOrganizationPluginsRequest, opts ...grpc.CallOption) (*ListOrganizationPluginsResponse, error) {
	out := new(ListOrganizationPluginsResponse)
	err := c.cc.Invoke(ctx, PluginService_ListOrganizationPlugins_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) GetPluginVersion(ctx context.Context, in *GetPluginVersionRequest, opts ...grpc.CallOption) (*GetPluginVersionResponse, error) {
	out := new(GetPluginVersionResponse)
	err := c.cc.Invoke(ctx, PluginService_GetPluginVersion_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListPluginVersions(ctx context.Context, in *ListPluginVersionsRequest, opts ...grpc.CallOption) (*ListPluginVersionsResponse, error) {
	out := new(ListPluginVersionsResponse)
	err := c.cc.Invoke(ctx, PluginService_ListPluginVersions_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) GetPlugin(ctx context.Context, in *GetPluginRequest, opts ...grpc.CallOption) (*GetPluginResponse, error) {
	out := new(GetPluginResponse)
	err := c.cc.Invoke(ctx, PluginService_GetPlugin_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) DeletePlugin(ctx context.Context, in *DeletePluginRequest, opts ...grpc.CallOption) (*DeletePluginResponse, error) {
	out := new(DeletePluginResponse)
	err := c.cc.Invoke(ctx, PluginService_DeletePlugin_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) GetTemplate(ctx context.Context, in *GetTemplateRequest, opts ...grpc.CallOption) (*GetTemplateResponse, error) {
	out := new(GetTemplateResponse)
	err := c.cc.Invoke(ctx, PluginService_GetTemplate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListTemplates(ctx context.Context, in *ListTemplatesRequest, opts ...grpc.CallOption) (*ListTemplatesResponse, error) {
	out := new(ListTemplatesResponse)
	err := c.cc.Invoke(ctx, PluginService_ListTemplates_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListTemplatesUserCanAccess(ctx context.Context, in *ListTemplatesUserCanAccessRequest, opts ...grpc.CallOption) (*ListTemplatesUserCanAccessResponse, error) {
	out := new(ListTemplatesUserCanAccessResponse)
	err := c.cc.Invoke(ctx, PluginService_ListTemplatesUserCanAccess_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListUserTemplates(ctx context.Context, in *ListUserTemplatesRequest, opts ...grpc.CallOption) (*ListUserTemplatesResponse, error) {
	out := new(ListUserTemplatesResponse)
	err := c.cc.Invoke(ctx, PluginService_ListUserTemplates_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListOrganizationTemplates(ctx context.Context, in *ListOrganizationTemplatesRequest, opts ...grpc.CallOption) (*ListOrganizationTemplatesResponse, error) {
	out := new(ListOrganizationTemplatesResponse)
	err := c.cc.Invoke(ctx, PluginService_ListOrganizationTemplates_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) GetTemplateVersion(ctx context.Context, in *GetTemplateVersionRequest, opts ...grpc.CallOption) (*GetTemplateVersionResponse, error) {
	out := new(GetTemplateVersionResponse)
	err := c.cc.Invoke(ctx, PluginService_GetTemplateVersion_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) ListTemplateVersions(ctx context.Context, in *ListTemplateVersionsRequest, opts ...grpc.CallOption) (*ListTemplateVersionsResponse, error) {
	out := new(ListTemplateVersionsResponse)
	err := c.cc.Invoke(ctx, PluginService_ListTemplateVersions_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginServiceClient) DeleteTemplate(ctx context.Context, in *DeleteTemplateRequest, opts ...grpc.CallOption) (*DeleteTemplateResponse, error) {
	out := new(DeleteTemplateResponse)
	err := c.cc.Invoke(ctx, PluginService_DeleteTemplate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PluginServiceServer is the server API for PluginService service.
// All implementations must embed UnimplementedPluginServiceServer
// for forward compatibility
type PluginServiceServer interface {
	// ListPlugins returns all the plugins available to the user. This includes
	// public plugins, those uploaded to organizations the user is part of,
	// and any plugins uploaded directly by the user.
	ListPlugins(context.Context, *ListPluginsRequest) (*ListPluginsResponse, error)
	// ListUserPlugins lists all plugins belonging to a user.
	ListUserPlugins(context.Context, *ListUserPluginsRequest) (*ListUserPluginsResponse, error)
	// ListOrganizationPlugins lists all plugins for an organization.
	ListOrganizationPlugins(context.Context, *ListOrganizationPluginsRequest) (*ListOrganizationPluginsResponse, error)
	// GetPluginVersion returns the plugin version, if found.
	GetPluginVersion(context.Context, *GetPluginVersionRequest) (*GetPluginVersionResponse, error)
	// ListPluginVersions lists all the versions available for the specified plugin.
	ListPluginVersions(context.Context, *ListPluginVersionsRequest) (*ListPluginVersionsResponse, error)
	// GetPlugin returns the plugin, if found.
	GetPlugin(context.Context, *GetPluginRequest) (*GetPluginResponse, error)
	// DeletePlugin deletes the plugin, if it exists. Note that deleting
	// a plugin may cause breaking changes for templates using that plugin,
	// and should be done with extreme care.
	DeletePlugin(context.Context, *DeletePluginRequest) (*DeletePluginResponse, error)
	// GetTemplate returns the template, if found.
	GetTemplate(context.Context, *GetTemplateRequest) (*GetTemplateResponse, error)
	// ListTemplates returns all the templates available to the user. This includes
	// public templates, those owned by organizations the user is part of,
	// and any created directly by the user.
	ListTemplates(context.Context, *ListTemplatesRequest) (*ListTemplatesResponse, error)
	// ListTemplatesUserCanAccess is like ListTemplates, but does not return
	// public templates.
	ListTemplatesUserCanAccess(context.Context, *ListTemplatesUserCanAccessRequest) (*ListTemplatesUserCanAccessResponse, error)
	// ListUserPlugins lists all templates belonging to a user.
	ListUserTemplates(context.Context, *ListUserTemplatesRequest) (*ListUserTemplatesResponse, error)
	// ListOrganizationTemplates lists all templates for an organization.
	ListOrganizationTemplates(context.Context, *ListOrganizationTemplatesRequest) (*ListOrganizationTemplatesResponse, error)
	// GetTemplateVersion returns the template version, if found.
	GetTemplateVersion(context.Context, *GetTemplateVersionRequest) (*GetTemplateVersionResponse, error)
	// ListTemplateVersions lists all the template versions available for the specified template.
	ListTemplateVersions(context.Context, *ListTemplateVersionsRequest) (*ListTemplateVersionsResponse, error)
	// DeleteTemplate deletes the template, if it exists.
	DeleteTemplate(context.Context, *DeleteTemplateRequest) (*DeleteTemplateResponse, error)
	mustEmbedUnimplementedPluginServiceServer()
}

// UnimplementedPluginServiceServer must be embedded to have forward compatible implementations.
type UnimplementedPluginServiceServer struct {
}

func (UnimplementedPluginServiceServer) ListPlugins(context.Context, *ListPluginsRequest) (*ListPluginsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPlugins not implemented")
}
func (UnimplementedPluginServiceServer) ListUserPlugins(context.Context, *ListUserPluginsRequest) (*ListUserPluginsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUserPlugins not implemented")
}
func (UnimplementedPluginServiceServer) ListOrganizationPlugins(context.Context, *ListOrganizationPluginsRequest) (*ListOrganizationPluginsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizationPlugins not implemented")
}
func (UnimplementedPluginServiceServer) GetPluginVersion(context.Context, *GetPluginVersionRequest) (*GetPluginVersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPluginVersion not implemented")
}
func (UnimplementedPluginServiceServer) ListPluginVersions(context.Context, *ListPluginVersionsRequest) (*ListPluginVersionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPluginVersions not implemented")
}
func (UnimplementedPluginServiceServer) GetPlugin(context.Context, *GetPluginRequest) (*GetPluginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPlugin not implemented")
}
func (UnimplementedPluginServiceServer) DeletePlugin(context.Context, *DeletePluginRequest) (*DeletePluginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeletePlugin not implemented")
}
func (UnimplementedPluginServiceServer) GetTemplate(context.Context, *GetTemplateRequest) (*GetTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTemplate not implemented")
}
func (UnimplementedPluginServiceServer) ListTemplates(context.Context, *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTemplates not implemented")
}
func (UnimplementedPluginServiceServer) ListTemplatesUserCanAccess(context.Context, *ListTemplatesUserCanAccessRequest) (*ListTemplatesUserCanAccessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTemplatesUserCanAccess not implemented")
}
func (UnimplementedPluginServiceServer) ListUserTemplates(context.Context, *ListUserTemplatesRequest) (*ListUserTemplatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUserTemplates not implemented")
}
func (UnimplementedPluginServiceServer) ListOrganizationTemplates(context.Context, *ListOrganizationTemplatesRequest) (*ListOrganizationTemplatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizationTemplates not implemented")
}
func (UnimplementedPluginServiceServer) GetTemplateVersion(context.Context, *GetTemplateVersionRequest) (*GetTemplateVersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTemplateVersion not implemented")
}
func (UnimplementedPluginServiceServer) ListTemplateVersions(context.Context, *ListTemplateVersionsRequest) (*ListTemplateVersionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTemplateVersions not implemented")
}
func (UnimplementedPluginServiceServer) DeleteTemplate(context.Context, *DeleteTemplateRequest) (*DeleteTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteTemplate not implemented")
}
func (UnimplementedPluginServiceServer) mustEmbedUnimplementedPluginServiceServer() {}

// UnsafePluginServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PluginServiceServer will
// result in compilation errors.
type UnsafePluginServiceServer interface {
	mustEmbedUnimplementedPluginServiceServer()
}

func RegisterPluginServiceServer(s grpc.ServiceRegistrar, srv PluginServiceServer) {
	s.RegisterService(&PluginService_ServiceDesc, srv)
}

func _PluginService_ListPlugins_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPluginsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListPlugins(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListPlugins_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListPlugins(ctx, req.(*ListPluginsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListUserPlugins_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserPluginsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListUserPlugins(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListUserPlugins_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListUserPlugins(ctx, req.(*ListUserPluginsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListOrganizationPlugins_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationPluginsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListOrganizationPlugins(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListOrganizationPlugins_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListOrganizationPlugins(ctx, req.(*ListOrganizationPluginsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_GetPluginVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPluginVersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetPluginVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetPluginVersion_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetPluginVersion(ctx, req.(*GetPluginVersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListPluginVersions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPluginVersionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListPluginVersions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListPluginVersions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListPluginVersions(ctx, req.(*ListPluginVersionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_GetPlugin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPluginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetPlugin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetPlugin_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetPlugin(ctx, req.(*GetPluginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_DeletePlugin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeletePluginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).DeletePlugin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_DeletePlugin_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).DeletePlugin(ctx, req.(*DeletePluginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_GetTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetTemplate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetTemplate(ctx, req.(*GetTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListTemplates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTemplatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListTemplates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListTemplates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListTemplates(ctx, req.(*ListTemplatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListTemplatesUserCanAccess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTemplatesUserCanAccessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListTemplatesUserCanAccess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListTemplatesUserCanAccess_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListTemplatesUserCanAccess(ctx, req.(*ListTemplatesUserCanAccessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListUserTemplates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserTemplatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListUserTemplates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListUserTemplates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListUserTemplates(ctx, req.(*ListUserTemplatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListOrganizationTemplates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationTemplatesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListOrganizationTemplates(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListOrganizationTemplates_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListOrganizationTemplates(ctx, req.(*ListOrganizationTemplatesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_GetTemplateVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTemplateVersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).GetTemplateVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_GetTemplateVersion_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).GetTemplateVersion(ctx, req.(*GetTemplateVersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_ListTemplateVersions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTemplateVersionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).ListTemplateVersions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_ListTemplateVersions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).ListTemplateVersions(ctx, req.(*ListTemplateVersionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PluginService_DeleteTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServiceServer).DeleteTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PluginService_DeleteTemplate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServiceServer).DeleteTemplate(ctx, req.(*DeleteTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PluginService_ServiceDesc is the grpc.ServiceDesc for PluginService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PluginService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "bufman.dubbo.apache.org.registry.v1alpha1.PluginService",
	HandlerType: (*PluginServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListPlugins",
			Handler:    _PluginService_ListPlugins_Handler,
		},
		{
			MethodName: "ListUserPlugins",
			Handler:    _PluginService_ListUserPlugins_Handler,
		},
		{
			MethodName: "ListOrganizationPlugins",
			Handler:    _PluginService_ListOrganizationPlugins_Handler,
		},
		{
			MethodName: "GetPluginVersion",
			Handler:    _PluginService_GetPluginVersion_Handler,
		},
		{
			MethodName: "ListPluginVersions",
			Handler:    _PluginService_ListPluginVersions_Handler,
		},
		{
			MethodName: "GetPlugin",
			Handler:    _PluginService_GetPlugin_Handler,
		},
		{
			MethodName: "DeletePlugin",
			Handler:    _PluginService_DeletePlugin_Handler,
		},
		{
			MethodName: "GetTemplate",
			Handler:    _PluginService_GetTemplate_Handler,
		},
		{
			MethodName: "ListTemplates",
			Handler:    _PluginService_ListTemplates_Handler,
		},
		{
			MethodName: "ListTemplatesUserCanAccess",
			Handler:    _PluginService_ListTemplatesUserCanAccess_Handler,
		},
		{
			MethodName: "ListUserTemplates",
			Handler:    _PluginService_ListUserTemplates_Handler,
		},
		{
			MethodName: "ListOrganizationTemplates",
			Handler:    _PluginService_ListOrganizationTemplates_Handler,
		},
		{
			MethodName: "GetTemplateVersion",
			Handler:    _PluginService_GetTemplateVersion_Handler,
		},
		{
			MethodName: "ListTemplateVersions",
			Handler:    _PluginService_ListTemplateVersions_Handler,
		},
		{
			MethodName: "DeleteTemplate",
			Handler:    _PluginService_DeleteTemplate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "registry/v1alpha1/plugin.proto",
}
