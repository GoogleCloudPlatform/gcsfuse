// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// GENERATED CODE - DO NOT EDIT MANUALLY.

syntax = "proto3";

package cfg;

option go_package = "github.com/googlecloudplatform/gcsfuse/v3/cfg/pb;pb";

{{range .TypeTemplateData}}
{{$typeName := .TypeName}}
message {{$typeName}} {
  {{- range $idx, $fld := .Fields}}
  {{protoType $fld.DataType}} {{protoFieldName $fld.FieldName}} = {{protoTag $typeName $fld.FieldName}};
  {{- end}}
}
{{end}}
