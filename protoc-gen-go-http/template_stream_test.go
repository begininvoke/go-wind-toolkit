package main

import (
	"strings"
	"testing"
)

// handlerSegment 截取指定 handler 函数体(到下一个顶层 func 为止)。
func handlerSegment(t *testing.T, out, marker string) string {
	t.Helper()
	start := strings.Index(out, marker)
	if start < 0 {
		t.Fatalf("handler %q not found in rendered output", marker)
	}
	seg := out[start:]
	if next := strings.Index(seg[len(marker):], "\nfunc "); next >= 0 {
		seg = seg[: len(marker)+next]
	}
	return seg
}

// 流式 methodDesc 经模板渲染的形态契约:
//   - 客户端流式:接口方法收 *http.Request(请求体即字节流),回包为普通消息;
//     handler 不做任何 proto 绑定,直接把请求交给服务并 WriteResponse 回包
//   - 服务端流式:接口方法收请求消息 + HttpBody emit 回调;handler 正常做
//     query/path 绑定,emit 经 binding.WriteStreamChunk 分块下发,首块前
//     出错才走 WriteError(开始流式后只中止)
//   - 普通方法:形态与既有完全一致
func TestStreamingTemplateShapes(t *testing.T) {
	sd := &serviceDesc{
		ServiceType: "Svc",
		ServiceName: "test.v1.Svc",
		Metadata:    "x.proto",
		Methods: []*methodDesc{
			{
				Name:            "Upload",
				OriginalName:    "Upload",
				Num:             0,
				Request:         "UploadRespPlaceholder",
				Reply:           "UploadResp",
				Path:            "/v1/upload",
				Method:          "POST",
				ClientStreaming: true,
			},
			{
				Name:            "Download",
				OriginalName:    "Download",
				Num:             0,
				Request:         "DownloadReq",
				Reply:           "DownloadRespPlaceholder",
				StreamElem:      "httpbody.HttpBody",
				Path:            "/v1/download",
				Method:          "GET",
				HasVars:         false,
				ServerStreaming: true,
			},
			{
				Name:         "Plain",
				OriginalName: "Plain",
				Num:          0,
				Request:      "PlainReq",
				Reply:        "PlainResp",
				Path:         "/v1/plain",
				Method:       "POST",
				HasBody:      true,
				BodyField:    "*",
			},
		},
	}
	out := sd.execute()

	// 接口形态
	for _, want := range []string{
		"Upload(context.Context, *http.Request) (*UploadResp, error)",
		"Download(context.Context, *DownloadReq, func(*httpbody.HttpBody) error) error",
		"Plain(context.Context, *PlainReq) (*PlainResp, error)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("interface missing %q", want)
		}
	}

	// 上行流式 handler:无绑定前缀(不得出现 var in)、请求原样直传、回包正常写出
	uploadSeg := handlerSegment(t, out, "func _Svc_Upload0_HTTP_Handler")
	if strings.Contains(uploadSeg, "var in") {
		t.Error("client-streaming handler must not bind a request message")
	}
	if !strings.Contains(uploadSeg, "out, err := svc.Upload(r.Context(), r)") {
		t.Error("client-streaming handler missing raw-request passthrough")
	}
	if !strings.Contains(uploadSeg, "binding.WriteResponse(w, r, out)") {
		t.Error("client-streaming handler missing WriteResponse for reply")
	}

	// 下行流式 handler:emit 闭包走 WriteStreamChunk,started 守卫
	if !strings.Contains(out, "binding.WriteStreamChunk(w, elem.GetContentType(), elem.GetData())") {
		t.Error("server-streaming handler missing WriteStreamChunk emit")
	}
	if !strings.Contains(out, "if !started {") {
		t.Error("server-streaming handler missing started guard")
	}
	dlSeg := handlerSegment(t, out, "func _Svc_Download0_HTTP_Handler")
	if !strings.Contains(dlSeg, "binding.BindQuery(&in, r.URL.Query())") {
		t.Error("server-streaming handler missing query binding")
	}

	// 普通方法不受影响
	plainSeg := handlerSegment(t, out, "func _Svc_Plain0_HTTP_Handler")
	if !strings.Contains(plainSeg, "binding.BindBody(r, &in)") {
		t.Error("plain handler missing body binding")
	}
	if !strings.Contains(plainSeg, "binding.WriteResponse(w, r, out)") {
		t.Error("plain handler missing WriteResponse")
	}
}
