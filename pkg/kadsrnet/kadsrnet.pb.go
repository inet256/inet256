// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.14.0
// source: kadsrnet.proto

package kadsrnet

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Message struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Src        []byte   `protobuf:"bytes,1,opt,name=src,proto3" json:"src,omitempty"`
	Dst        []byte   `protobuf:"bytes,2,opt,name=dst,proto3" json:"dst,omitempty"`
	Path       []uint32 `protobuf:"varint,3,rep,packed,name=path,proto3" json:"path,omitempty"`
	PathMatch  uint32   `protobuf:"varint,8,opt,name=path_match,json=pathMatch,proto3" json:"path_match,omitempty"`
	ReturnPath []uint32 `protobuf:"varint,4,rep,packed,name=return_path,json=returnPath,proto3" json:"return_path,omitempty"`
	Body       []byte   `protobuf:"bytes,5,opt,name=body,proto3" json:"body,omitempty"`
	Timestamp  int64    `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Sig        []byte   `protobuf:"bytes,7,opt,name=sig,proto3" json:"sig,omitempty"`
}

func (x *Message) Reset() {
	*x = Message{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Message) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Message) ProtoMessage() {}

func (x *Message) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Message.ProtoReflect.Descriptor instead.
func (*Message) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{0}
}

func (x *Message) GetSrc() []byte {
	if x != nil {
		return x.Src
	}
	return nil
}

func (x *Message) GetDst() []byte {
	if x != nil {
		return x.Dst
	}
	return nil
}

func (x *Message) GetPath() []uint32 {
	if x != nil {
		return x.Path
	}
	return nil
}

func (x *Message) GetPathMatch() uint32 {
	if x != nil {
		return x.PathMatch
	}
	return 0
}

func (x *Message) GetReturnPath() []uint32 {
	if x != nil {
		return x.ReturnPath
	}
	return nil
}

func (x *Message) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

func (x *Message) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *Message) GetSig() []byte {
	if x != nil {
		return x.Sig
	}
	return nil
}

type Body struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Body:
	//	*Body_Data
	//	*Body_QueryRoutes
	//	*Body_RouteList
	//	*Body_PeerInfoReq
	//	*Body_PeerInfo
	//	*Body_Ping
	//	*Body_Pong
	Body isBody_Body `protobuf_oneof:"body"`
}

func (x *Body) Reset() {
	*x = Body{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Body) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Body) ProtoMessage() {}

func (x *Body) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Body.ProtoReflect.Descriptor instead.
func (*Body) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{1}
}

func (m *Body) GetBody() isBody_Body {
	if m != nil {
		return m.Body
	}
	return nil
}

func (x *Body) GetData() []byte {
	if x, ok := x.GetBody().(*Body_Data); ok {
		return x.Data
	}
	return nil
}

func (x *Body) GetQueryRoutes() *QueryRoutes {
	if x, ok := x.GetBody().(*Body_QueryRoutes); ok {
		return x.QueryRoutes
	}
	return nil
}

func (x *Body) GetRouteList() *RouteList {
	if x, ok := x.GetBody().(*Body_RouteList); ok {
		return x.RouteList
	}
	return nil
}

func (x *Body) GetPeerInfoReq() *PeerInfoReq {
	if x, ok := x.GetBody().(*Body_PeerInfoReq); ok {
		return x.PeerInfoReq
	}
	return nil
}

func (x *Body) GetPeerInfo() *PeerInfo {
	if x, ok := x.GetBody().(*Body_PeerInfo); ok {
		return x.PeerInfo
	}
	return nil
}

func (x *Body) GetPing() *Ping {
	if x, ok := x.GetBody().(*Body_Ping); ok {
		return x.Ping
	}
	return nil
}

func (x *Body) GetPong() *Pong {
	if x, ok := x.GetBody().(*Body_Pong); ok {
		return x.Pong
	}
	return nil
}

type isBody_Body interface {
	isBody_Body()
}

type Body_Data struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3,oneof"`
}

type Body_QueryRoutes struct {
	QueryRoutes *QueryRoutes `protobuf:"bytes,3,opt,name=query_routes,json=queryRoutes,proto3,oneof"`
}

type Body_RouteList struct {
	RouteList *RouteList `protobuf:"bytes,4,opt,name=route_list,json=routeList,proto3,oneof"`
}

type Body_PeerInfoReq struct {
	PeerInfoReq *PeerInfoReq `protobuf:"bytes,5,opt,name=peer_info_req,json=peerInfoReq,proto3,oneof"`
}

type Body_PeerInfo struct {
	PeerInfo *PeerInfo `protobuf:"bytes,6,opt,name=peer_info,json=peerInfo,proto3,oneof"`
}

type Body_Ping struct {
	Ping *Ping `protobuf:"bytes,7,opt,name=ping,proto3,oneof"`
}

type Body_Pong struct {
	Pong *Pong `protobuf:"bytes,8,opt,name=pong,proto3,oneof"`
}

func (*Body_Data) isBody_Body() {}

func (*Body_QueryRoutes) isBody_Body() {}

func (*Body_RouteList) isBody_Body() {}

func (*Body_PeerInfoReq) isBody_Body() {}

func (*Body_PeerInfo) isBody_Body() {}

func (*Body_Ping) isBody_Body() {}

func (*Body_Pong) isBody_Body() {}

type PeerInfoReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *PeerInfoReq) Reset() {
	*x = PeerInfoReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PeerInfoReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerInfoReq) ProtoMessage() {}

func (x *PeerInfoReq) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerInfoReq.ProtoReflect.Descriptor instead.
func (*PeerInfoReq) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{2}
}

type PeerInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PublicKey []byte `protobuf:"bytes,2,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
}

func (x *PeerInfo) Reset() {
	*x = PeerInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PeerInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerInfo) ProtoMessage() {}

func (x *PeerInfo) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerInfo.ProtoReflect.Descriptor instead.
func (*PeerInfo) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{3}
}

func (x *PeerInfo) GetPublicKey() []byte {
	if x != nil {
		return x.PublicKey
	}
	return nil
}

type QueryRoutes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Locus []byte `protobuf:"bytes,1,opt,name=locus,proto3" json:"locus,omitempty"`
	Nbits uint32 `protobuf:"varint,2,opt,name=nbits,proto3" json:"nbits,omitempty"`
	Limit uint32 `protobuf:"varint,3,opt,name=limit,proto3" json:"limit,omitempty"`
}

func (x *QueryRoutes) Reset() {
	*x = QueryRoutes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryRoutes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRoutes) ProtoMessage() {}

func (x *QueryRoutes) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRoutes.ProtoReflect.Descriptor instead.
func (*QueryRoutes) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{4}
}

func (x *QueryRoutes) GetLocus() []byte {
	if x != nil {
		return x.Locus
	}
	return nil
}

func (x *QueryRoutes) GetNbits() uint32 {
	if x != nil {
		return x.Nbits
	}
	return 0
}

func (x *QueryRoutes) GetLimit() uint32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

type Route struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dst       []byte   `protobuf:"bytes,1,opt,name=dst,proto3" json:"dst,omitempty"`
	Path      []uint32 `protobuf:"varint,2,rep,packed,name=path,proto3" json:"path,omitempty"`
	Timestamp int64    `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *Route) Reset() {
	*x = Route{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Route) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Route) ProtoMessage() {}

func (x *Route) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Route.ProtoReflect.Descriptor instead.
func (*Route) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{5}
}

func (x *Route) GetDst() []byte {
	if x != nil {
		return x.Dst
	}
	return nil
}

func (x *Route) GetPath() []uint32 {
	if x != nil {
		return x.Path
	}
	return nil
}

func (x *Route) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type RouteList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Routes []*Route `protobuf:"bytes,1,rep,name=routes,proto3" json:"routes,omitempty"`
	Src    []byte   `protobuf:"bytes,2,opt,name=src,proto3" json:"src,omitempty"`
}

func (x *RouteList) Reset() {
	*x = RouteList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RouteList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RouteList) ProtoMessage() {}

func (x *RouteList) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RouteList.ProtoReflect.Descriptor instead.
func (*RouteList) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{6}
}

func (x *RouteList) GetRoutes() []*Route {
	if x != nil {
		return x.Routes
	}
	return nil
}

func (x *RouteList) GetSrc() []byte {
	if x != nil {
		return x.Src
	}
	return nil
}

type Ping struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uuid      []byte `protobuf:"bytes,1,opt,name=uuid,proto3" json:"uuid,omitempty"`
	Timestamp int64  `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *Ping) Reset() {
	*x = Ping{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ping) ProtoMessage() {}

func (x *Ping) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ping.ProtoReflect.Descriptor instead.
func (*Ping) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{7}
}

func (x *Ping) GetUuid() []byte {
	if x != nil {
		return x.Uuid
	}
	return nil
}

func (x *Ping) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type Pong struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ping *Ping `protobuf:"bytes,1,opt,name=ping,proto3" json:"ping,omitempty"`
}

func (x *Pong) Reset() {
	*x = Pong{}
	if protoimpl.UnsafeEnabled {
		mi := &file_kadsrnet_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Pong) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pong) ProtoMessage() {}

func (x *Pong) ProtoReflect() protoreflect.Message {
	mi := &file_kadsrnet_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pong.ProtoReflect.Descriptor instead.
func (*Pong) Descriptor() ([]byte, []int) {
	return file_kadsrnet_proto_rawDescGZIP(), []int{8}
}

func (x *Pong) GetPing() *Ping {
	if x != nil {
		return x.Ping
	}
	return nil
}

var File_kadsrnet_proto protoreflect.FileDescriptor

var file_kadsrnet_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x6b, 0x61, 0x64, 0x73, 0x72, 0x6e, 0x65, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xc5, 0x01, 0x0a, 0x07, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x10, 0x0a, 0x03,
	0x73, 0x72, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x73, 0x72, 0x63, 0x12, 0x10,
	0x0a, 0x03, 0x64, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x64, 0x73, 0x74,
	0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x04,
	0x70, 0x61, 0x74, 0x68, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x61, 0x74, 0x68, 0x5f, 0x6d, 0x61, 0x74,
	0x63, 0x68, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x70, 0x61, 0x74, 0x68, 0x4d, 0x61,
	0x74, 0x63, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x5f, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x0a, 0x72, 0x65, 0x74, 0x75, 0x72, 0x6e,
	0x50, 0x61, 0x74, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x69, 0x67, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x03, 0x73, 0x69, 0x67, 0x22, 0x9c, 0x02, 0x0a, 0x04, 0x42, 0x6f, 0x64,
	0x79, 0x12, 0x14, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x48,
	0x00, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x31, 0x0a, 0x0c, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x5f, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e,
	0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x48, 0x00, 0x52, 0x0b, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x12, 0x2b, 0x0a, 0x0a, 0x72, 0x6f,
	0x75, 0x74, 0x65, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a,
	0x2e, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x48, 0x00, 0x52, 0x09, 0x72, 0x6f,
	0x75, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x32, 0x0a, 0x0d, 0x70, 0x65, 0x65, 0x72, 0x5f,
	0x69, 0x6e, 0x66, 0x6f, 0x5f, 0x72, 0x65, 0x71, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c,
	0x2e, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x48, 0x00, 0x52, 0x0b,
	0x70, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x12, 0x28, 0x0a, 0x09, 0x70,
	0x65, 0x65, 0x72, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09,
	0x2e, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x48, 0x00, 0x52, 0x08, 0x70, 0x65, 0x65,
	0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1b, 0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x05, 0x2e, 0x50, 0x69, 0x6e, 0x67, 0x48, 0x00, 0x52, 0x04, 0x70, 0x69,
	0x6e, 0x67, 0x12, 0x1b, 0x0a, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x05, 0x2e, 0x50, 0x6f, 0x6e, 0x67, 0x48, 0x00, 0x52, 0x04, 0x70, 0x6f, 0x6e, 0x67, 0x42,
	0x06, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x22, 0x0d, 0x0a, 0x0b, 0x50, 0x65, 0x65, 0x72, 0x49,
	0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x22, 0x29, 0x0a, 0x08, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e,
	0x66, 0x6f, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65,
	0x79, 0x22, 0x4f, 0x0a, 0x0b, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x73,
	0x12, 0x14, 0x0a, 0x05, 0x6c, 0x6f, 0x63, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x05, 0x6c, 0x6f, 0x63, 0x75, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x62, 0x69, 0x74, 0x73, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x6e, 0x62, 0x69, 0x74, 0x73, 0x12, 0x14, 0x0a, 0x05,
	0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x6c, 0x69, 0x6d,
	0x69, 0x74, 0x22, 0x4b, 0x0a, 0x05, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x64,
	0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x64, 0x73, 0x74, 0x12, 0x12, 0x0a,
	0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x04, 0x70, 0x61, 0x74,
	0x68, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22,
	0x3d, 0x0a, 0x09, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x1e, 0x0a, 0x06,
	0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x06, 0x2e, 0x52,
	0x6f, 0x75, 0x74, 0x65, 0x52, 0x06, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x12, 0x10, 0x0a, 0x03,
	0x73, 0x72, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x73, 0x72, 0x63, 0x22, 0x38,
	0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x75, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x75, 0x75, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x21, 0x0a, 0x04, 0x50, 0x6f, 0x6e, 0x67,
	0x12, 0x19, 0x0a, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x05,
	0x2e, 0x50, 0x69, 0x6e, 0x67, 0x52, 0x04, 0x70, 0x69, 0x6e, 0x67, 0x42, 0x29, 0x5a, 0x27, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x69, 0x6e, 0x65, 0x74, 0x32, 0x35,
	0x36, 0x2f, 0x69, 0x6e, 0x65, 0x74, 0x32, 0x35, 0x36, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x6b, 0x61,
	0x64, 0x73, 0x72, 0x6e, 0x65, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_kadsrnet_proto_rawDescOnce sync.Once
	file_kadsrnet_proto_rawDescData = file_kadsrnet_proto_rawDesc
)

func file_kadsrnet_proto_rawDescGZIP() []byte {
	file_kadsrnet_proto_rawDescOnce.Do(func() {
		file_kadsrnet_proto_rawDescData = protoimpl.X.CompressGZIP(file_kadsrnet_proto_rawDescData)
	})
	return file_kadsrnet_proto_rawDescData
}

var file_kadsrnet_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_kadsrnet_proto_goTypes = []interface{}{
	(*Message)(nil),     // 0: Message
	(*Body)(nil),        // 1: Body
	(*PeerInfoReq)(nil), // 2: PeerInfoReq
	(*PeerInfo)(nil),    // 3: PeerInfo
	(*QueryRoutes)(nil), // 4: QueryRoutes
	(*Route)(nil),       // 5: Route
	(*RouteList)(nil),   // 6: RouteList
	(*Ping)(nil),        // 7: Ping
	(*Pong)(nil),        // 8: Pong
}
var file_kadsrnet_proto_depIdxs = []int32{
	4, // 0: Body.query_routes:type_name -> QueryRoutes
	6, // 1: Body.route_list:type_name -> RouteList
	2, // 2: Body.peer_info_req:type_name -> PeerInfoReq
	3, // 3: Body.peer_info:type_name -> PeerInfo
	7, // 4: Body.ping:type_name -> Ping
	8, // 5: Body.pong:type_name -> Pong
	5, // 6: RouteList.routes:type_name -> Route
	7, // 7: Pong.ping:type_name -> Ping
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_kadsrnet_proto_init() }
func file_kadsrnet_proto_init() {
	if File_kadsrnet_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_kadsrnet_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Message); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Body); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PeerInfoReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PeerInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryRoutes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Route); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RouteList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ping); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_kadsrnet_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Pong); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_kadsrnet_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Body_Data)(nil),
		(*Body_QueryRoutes)(nil),
		(*Body_RouteList)(nil),
		(*Body_PeerInfoReq)(nil),
		(*Body_PeerInfo)(nil),
		(*Body_Ping)(nil),
		(*Body_Pong)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_kadsrnet_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_kadsrnet_proto_goTypes,
		DependencyIndexes: file_kadsrnet_proto_depIdxs,
		MessageInfos:      file_kadsrnet_proto_msgTypes,
	}.Build()
	File_kadsrnet_proto = out.File
	file_kadsrnet_proto_rawDesc = nil
	file_kadsrnet_proto_goTypes = nil
	file_kadsrnet_proto_depIdxs = nil
}
