// Copyright 2022 Google Inc. All Rights Reserved.
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

package storageutil

import (
	"crypto/md5"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
	storagev1 "google.golang.org/api/storage/v1"
)

const TestBucketName string = "gcsfuse-default-bucket"
const TestObjectName string = "gcsfuse/default.txt"

func TestObjectAttrs(t *testing.T) { RunTests(t) }

type objectAttrsTest struct {
}

func init() { RegisterTestSuite(&objectAttrsTest{}) }

func (t objectAttrsTest) TestConvertACLRuleToObjectAccessControlMethod() {
	var attrs = storage.ACLRule{
		Entity:   "allUsers",
		EntityID: "123",
		Role:     "OWNER",
		Domain:   "Domain",
		Email:    "Email",
		ProjectTeam: &storage.ProjectTeam{
			ProjectNumber: "123",
			Team:          "Team",
		},
	}

	objectAccessControl := convertACLRuleToObjectAccessControl(attrs)

	ExpectEq(objectAccessControl.Entity, string(attrs.Entity))
	ExpectEq(objectAccessControl.EntityId, attrs.EntityID)
	ExpectEq(objectAccessControl.Role, string(attrs.Role))
	ExpectEq(objectAccessControl.Domain, attrs.Domain)
	ExpectEq(objectAccessControl.Email, attrs.Email)
	ExpectEq(objectAccessControl.ProjectTeam.ProjectNumber, attrs.ProjectTeam.ProjectNumber)
	ExpectEq(objectAccessControl.ProjectTeam.Team, attrs.ProjectTeam.Team)
}

func (t objectAttrsTest) TestConvertACLRuleToObjectAccessControlMethodWhenProjectTeamEqualsNil() {
	var attrs = storage.ACLRule{
		ProjectTeam: nil,
	}

	objectAccessControl := convertACLRuleToObjectAccessControl(attrs)

	ExpectEq(nil, objectAccessControl.ProjectTeam)
}

func (t objectAttrsTest) TestObjectAttrsToBucketObjectMethod() {
	var attrMd5 []byte
	timeAttr := time.Now()
	attrs := storage.ObjectAttrs{
		Bucket:                  TestBucketName,
		Name:                    TestObjectName,
		ContentType:             "ContentType",
		ContentLanguage:         "ContentLanguage",
		CacheControl:            "CacheControl",
		EventBasedHold:          true,
		TemporaryHold:           true,
		RetentionExpirationTime: timeAttr,
		ACL:                     nil,
		PredefinedACL:           "PredefinedACL",
		Owner:                   "Owner",
		Size:                    16,
		ContentEncoding:         "ContentEncoding",
		ContentDisposition:      "ContentDisposition",
		MD5:                     attrMd5,
		CRC32C:                  0,
		MediaLink:               "MediaLink",
		Metadata:                nil,
		Generation:              780,
		Metageneration:          0,
		StorageClass:            "StorageClass",
		Created:                 timeAttr,
		Deleted:                 timeAttr,
		Updated:                 timeAttr,
		CustomerKeySHA256:       "CustomerKeySHA256",
		KMSKeyName:              "KMSKeyName",
		Prefix:                  "Prefix",
		Etag:                    "Etag",
		CustomTime:              timeAttr,
		ComponentCount:          7,
	}
	customeTimeExpected := string(attrs.CustomTime.Format(time.RFC3339))

	var md5Expected [md5.Size]byte
	copy(md5Expected[:], attrs.MD5)

	var acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		acl = append(acl, convertACLRuleToObjectAccessControl(element))
	}

	object := ObjectAttrsToBucketObject(&attrs)

	ExpectEq(object.Name, attrs.Name)
	ExpectEq(object.ContentType, attrs.ContentType)
	ExpectEq(object.ContentLanguage, attrs.ContentLanguage)
	ExpectEq(object.CacheControl, attrs.CacheControl)
	ExpectEq(object.Owner, attrs.Owner)
	ExpectEq(object.Size, attrs.Size)
	ExpectEq(object.ContentEncoding, attrs.ContentEncoding)
	ExpectEq(len(object.MD5), len(&md5Expected))
	ExpectEq(cap(object.MD5), cap(&md5Expected))
	ExpectEq(*object.CRC32C, attrs.CRC32C)
	ExpectEq(object.MediaLink, attrs.MediaLink)
	ExpectEq(object.Metadata, attrs.Metadata)
	ExpectEq(object.Generation, attrs.Generation)
	ExpectEq(object.MetaGeneration, attrs.Metageneration)
	ExpectEq(object.StorageClass, attrs.StorageClass)
	ExpectEq(object.Updated.String(), attrs.Updated.String())
	ExpectEq(object.Deleted.String(), attrs.Deleted.String())
	ExpectEq(object.ContentDisposition, attrs.ContentDisposition)
	ExpectEq(object.CustomTime, customeTimeExpected)
	ExpectEq(object.EventBasedHold, attrs.EventBasedHold)
	ExpectEq(object.Acl, acl)
	ExpectEq(object.ComponentCount, attrs.ComponentCount)
}

func (t objectAttrsTest) TestConvertObjectAccessControlToACLRuleMethod() {
	objectAccessControl := &storagev1.ObjectAccessControl{
		Entity:   "test_entity",
		EntityId: "test_entity_id",
		Role:     "owner",
		Domain:   "test_domain",
		Email:    "test_email@test.com",
		ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
			ProjectNumber: "test_project_num",
			Team:          "test_team",
		},
	}

	aclRule := convertObjectAccessControlToACLRule(objectAccessControl)

	ExpectEq(aclRule.Entity, objectAccessControl.Entity)
	ExpectEq(aclRule.EntityID, objectAccessControl.EntityId)
	ExpectEq(aclRule.Role, objectAccessControl.Role)
	ExpectEq(aclRule.Domain, objectAccessControl.Domain)
	ExpectEq(aclRule.Email, objectAccessControl.Email)
	ExpectEq(aclRule.ProjectTeam.ProjectNumber, objectAccessControl.ProjectTeam.ProjectNumber)
	ExpectEq(aclRule.ProjectTeam.Team, objectAccessControl.ProjectTeam.Team)
}

func (t objectAttrsTest) TestConvertObjectAccessControlToACLRuleMethodWhenProjectTeamEqualsNil() {
	objectAccessControl := &storagev1.ObjectAccessControl{
		ProjectTeam: nil,
	}

	aclRule := convertObjectAccessControlToACLRule(objectAccessControl)

	ExpectEq(nil, aclRule.ProjectTeam)
}

func (t objectAttrsTest) TestSetAttrsInWriterMethod() {
	var crc32c uint32 = 45
	var generationPrecondition int64 = 3
	var metaGenerationPrecondition int64 = 33
	md5Hash := md5.Sum([]byte("testing"))
	timeInRFC3339 := "2006-01-02T15:04:05Z07:00"
	createObjectRequest := gcs.CreateObjectRequest{
		Name:                       "test_object",
		ContentType:                "json",
		ContentEncoding:            "universal",
		CacheControl:               "Medium",
		Metadata:                   map[string]string{"file_name": "test.txt"},
		ContentDisposition:         "Test content disposition",
		CustomTime:                 timeInRFC3339,
		EventBasedHold:             true,
		StorageClass:               "High Accessibility",
		Acl:                        nil,
		Contents:                   strings.NewReader("Creating new object"),
		CRC32C:                     &crc32c,
		MD5:                        &md5Hash,
		GenerationPrecondition:     &generationPrecondition,
		MetaGenerationPrecondition: &metaGenerationPrecondition,
	}
	writer := &storage.Writer{}

	writer = SetAttrsInWriter(writer, &createObjectRequest)

	ExpectEq(writer.Name, createObjectRequest.Name)
	ExpectEq(writer.ContentType, createObjectRequest.ContentType)
	ExpectEq(writer.ContentLanguage, createObjectRequest.ContentLanguage)
	ExpectEq(writer.ContentEncoding, createObjectRequest.ContentEncoding)
	ExpectEq(writer.CacheControl, createObjectRequest.CacheControl)
	ExpectEq(writer.Metadata, createObjectRequest.Metadata)
	ExpectEq(writer.ContentDisposition, createObjectRequest.ContentDisposition)
	parsedTime, _ := time.Parse(time.RFC3339, createObjectRequest.CustomTime)
	ExpectTrue(parsedTime.Equal(writer.CustomTime))
	ExpectEq(writer.EventBasedHold, createObjectRequest.EventBasedHold)
	ExpectEq(writer.StorageClass, createObjectRequest.StorageClass)
	ExpectEq(writer.CRC32C, *createObjectRequest.CRC32C)
	ExpectTrue(writer.SendCRC32C)
	ExpectEq(string(writer.MD5[:]), string(createObjectRequest.MD5[:]))
}

func (t objectAttrsTest) Test_ConvertObjToMinObject_WithNilObject() {
	var gcsObject *gcs.Object

	gcsMinObject := ConvertObjToMinObject(gcsObject)

	ExpectEq(nil, gcsMinObject)
}

func (t objectAttrsTest) Test_ConvertObjToMinObject_WithValidObject() {
	name := "test"
	size := uint64(36)
	generation := int64(444)
	metaGeneration := int64(555)
	currentTime := time.Now()
	contentEncode := "test_encoding"
	metadata := map[string]string{"test_key": "test_value"}
	gcsObject := gcs.Object{
		Name:            name,
		Size:            size,
		Generation:      generation,
		MetaGeneration:  metaGeneration,
		Updated:         currentTime,
		Metadata:        metadata,
		ContentEncoding: contentEncode,
	}

	gcsMinObject := ConvertObjToMinObject(&gcsObject)

	AssertNe(nil, gcsMinObject)
	ExpectEq(name, gcsMinObject.Name)
	ExpectEq(size, gcsMinObject.Size)
	ExpectEq(generation, gcsMinObject.Generation)
	ExpectEq(metaGeneration, gcsMinObject.MetaGeneration)
	ExpectTrue(currentTime.Equal(gcsMinObject.Updated))
	ExpectEq(contentEncode, gcsMinObject.ContentEncoding)
	ExpectEq(metadata, gcsMinObject.Metadata)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithNilObject() {
	var gcsObject *gcs.Object

	extendedObjAttr := ConvertObjToExtendedObjectAttributes(gcsObject)

	ExpectEq(nil, extendedObjAttr)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithValidObject() {
	var attrMd5 *[16]byte
	var crc32C uint32 = 0
	timeAttr := time.Now()
	gcsObject := gcs.Object{
		ContentType:        "ContentType",
		ContentLanguage:    "ContentLanguage",
		CacheControl:       "CacheControl",
		Owner:              "Owner",
		MD5:                attrMd5,
		CRC32C:             &crc32C,
		MediaLink:          "MediaLink",
		StorageClass:       "StorageClass",
		Deleted:            timeAttr,
		ComponentCount:     7,
		ContentDisposition: "ContentDisposition",
		CustomTime:         timeAttr.String(),
		EventBasedHold:     true,
		Acl:                nil,
	}

	extendedObjAttr := ConvertObjToExtendedObjectAttributes(&gcsObject)

	AssertNe(nil, extendedObjAttr)
	ExpectEq(gcsObject.ContentType, extendedObjAttr.ContentType)
	ExpectEq(gcsObject.ContentLanguage, extendedObjAttr.ContentLanguage)
	ExpectEq(gcsObject.CacheControl, extendedObjAttr.CacheControl)
	ExpectEq(gcsObject.Owner, extendedObjAttr.Owner)
	ExpectEq(gcsObject.MD5, extendedObjAttr.MD5)
	ExpectEq(gcsObject.CRC32C, extendedObjAttr.CRC32C)
	ExpectEq(gcsObject.MediaLink, extendedObjAttr.MediaLink)
	ExpectEq(gcsObject.StorageClass, extendedObjAttr.StorageClass)
	ExpectEq(0, gcsObject.Deleted.Compare(extendedObjAttr.Deleted))
	ExpectEq(gcsObject.ComponentCount, extendedObjAttr.ComponentCount)
	ExpectEq(gcsObject.ContentDisposition, extendedObjAttr.ContentDisposition)
	ExpectEq(gcsObject.CustomTime, extendedObjAttr.CustomTime)
	ExpectEq(gcsObject.EventBasedHold, extendedObjAttr.EventBasedHold)
	ExpectEq(gcsObject.Acl, extendedObjAttr.Acl)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithNilMinObjectAndNilAttributes() {
	var minObject *gcs.MinObject
	var extendedObjectAttr *gcs.ExtendedObjectAttributes

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	ExpectEq(nil, object)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithNilMinObjectAndNonNilAttributes() {
	var minObject *gcs.MinObject
	extendedObjectAttr := &gcs.ExtendedObjectAttributes{
		ContentType: "ContentType",
	}

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	ExpectEq(nil, object)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithNonNilMinObjectAndNilAttributes() {
	name := "test"
	minObject := &gcs.MinObject{
		Name: name,
	}
	var extendedObjectAttr *gcs.ExtendedObjectAttributes

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	ExpectEq(nil, object)
}

func (t objectAttrsTest) Test_ConvertObjToExtendedObjectAttributes_WithNonNilMinObjectAndNonNilAttributes() {
	var attrMd5 *[16]byte
	var crc32C uint32 = 0
	timeAttr := time.Now()
	minObject := &gcs.MinObject{
		Name:            "test",
		Size:            uint64(36),
		Generation:      int64(444),
		MetaGeneration:  int64(555),
		Updated:         timeAttr,
		Metadata:        map[string]string{"test_key": "test_value"},
		ContentEncoding: "test_encoding",
	}
	extendedObjAttr := &gcs.ExtendedObjectAttributes{
		ContentType:        "ContentType",
		ContentLanguage:    "ContentLanguage",
		CacheControl:       "CacheControl",
		Owner:              "Owner",
		MD5:                attrMd5,
		CRC32C:             &crc32C,
		MediaLink:          "MediaLink",
		StorageClass:       "StorageClass",
		Deleted:            timeAttr,
		ComponentCount:     7,
		ContentDisposition: "ContentDisposition",
		CustomTime:         timeAttr.String(),
		EventBasedHold:     true,
		Acl:                nil,
	}

	gcsObject := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjAttr)

	AssertNe(nil, gcsObject)
	ExpectEq(gcsObject.Name, minObject.Name)
	ExpectEq(gcsObject.Size, minObject.Size)
	ExpectEq(gcsObject.Generation, minObject.Generation)
	ExpectEq(gcsObject.MetaGeneration, minObject.MetaGeneration)
	ExpectEq(0, gcsObject.Updated.Compare(minObject.Updated))
	ExpectEq(gcsObject.Metadata, minObject.Metadata)
	ExpectEq(gcsObject.ContentEncoding, minObject.ContentEncoding)
	ExpectEq(gcsObject.ContentType, extendedObjAttr.ContentType)
	ExpectEq(gcsObject.ContentLanguage, extendedObjAttr.ContentLanguage)
	ExpectEq(gcsObject.CacheControl, extendedObjAttr.CacheControl)
	ExpectEq(gcsObject.Owner, extendedObjAttr.Owner)
	ExpectEq(gcsObject.MD5, extendedObjAttr.MD5)
	ExpectEq(gcsObject.CRC32C, extendedObjAttr.CRC32C)
	ExpectEq(gcsObject.MediaLink, extendedObjAttr.MediaLink)
	ExpectEq(gcsObject.StorageClass, extendedObjAttr.StorageClass)
	ExpectEq(0, gcsObject.Deleted.Compare(extendedObjAttr.Deleted))
	ExpectEq(gcsObject.ComponentCount, extendedObjAttr.ComponentCount)
	ExpectEq(gcsObject.ContentDisposition, extendedObjAttr.ContentDisposition)
	ExpectEq(gcsObject.CustomTime, extendedObjAttr.CustomTime)
	ExpectEq(gcsObject.EventBasedHold, extendedObjAttr.EventBasedHold)
	ExpectEq(gcsObject.Acl, extendedObjAttr.Acl)
}

func (t objectAttrsTest) Test_ConvertMinObjectToObject_WithNilMinObject() {
	var minObject *gcs.MinObject

	object := ConvertMinObjectToObject(minObject)

	ExpectEq(nil, object)
}

func (t objectAttrsTest) Test_ConvertMinObjectToObject_WithNonNilMinObject() {
	var attrMd5 *[16]byte
	var crc32C *uint32 = nil
	timeAttr := time.Now()
	minObject := &gcs.MinObject{
		Name:            "test",
		Size:            uint64(36),
		Generation:      int64(444),
		MetaGeneration:  int64(555),
		Updated:         timeAttr,
		Metadata:        map[string]string{"test_key": "test_value"},
		ContentEncoding: "test_encoding",
	}

	gcsObject := ConvertMinObjectToObject(minObject)

	AssertNe(nil, gcsObject)
	ExpectEq(gcsObject.Name, minObject.Name)
	ExpectEq(gcsObject.Size, minObject.Size)
	ExpectEq(gcsObject.Generation, minObject.Generation)
	ExpectEq(gcsObject.MetaGeneration, minObject.MetaGeneration)
	ExpectEq(0, gcsObject.Updated.Compare(minObject.Updated))
	ExpectEq(gcsObject.Metadata, minObject.Metadata)
	ExpectEq(gcsObject.ContentEncoding, minObject.ContentEncoding)
	ExpectEq(gcsObject.ContentType, "")
	ExpectEq(gcsObject.ContentLanguage, "")
	ExpectEq(gcsObject.CacheControl, "")
	ExpectEq(gcsObject.Owner, "")
	ExpectEq(gcsObject.MD5, attrMd5)
	ExpectEq(gcsObject.CRC32C, crc32C)
	ExpectEq(gcsObject.MediaLink, "")
	ExpectEq(gcsObject.StorageClass, "")
	ExpectEq(0, gcsObject.Deleted.Compare(time.Time{}))
	ExpectEq(gcsObject.ComponentCount, 0)
	ExpectEq(gcsObject.ContentDisposition, "")
	ExpectEq(gcsObject.CustomTime, "")
	ExpectEq(gcsObject.EventBasedHold, false)
	ExpectEq(gcsObject.Acl, []*storagev1.ObjectAccessControl(nil))
}
