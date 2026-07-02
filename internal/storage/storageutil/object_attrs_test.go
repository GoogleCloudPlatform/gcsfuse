// Copyright 2022 Google LLC
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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storagev1 "google.golang.org/api/storage/v1"
)

const TestBucketName string = "gcsfuse-default-bucket"
const TestObjectName string = "gcsfuse/default.txt"

func TestConvertACLRuleToObjectAccessControlMethod(t *testing.T) {
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

	assert.Equal(t, string(attrs.Entity), objectAccessControl.Entity)
	assert.Equal(t, attrs.EntityID, objectAccessControl.EntityId)
	assert.Equal(t, string(attrs.Role), objectAccessControl.Role)
	assert.Equal(t, attrs.Domain, objectAccessControl.Domain)
	assert.Equal(t, attrs.Email, objectAccessControl.Email)
	assert.Equal(t, attrs.ProjectTeam.ProjectNumber, objectAccessControl.ProjectTeam.ProjectNumber)
	assert.Equal(t, attrs.ProjectTeam.Team, objectAccessControl.ProjectTeam.Team)
}

func TestConvertACLRuleToObjectAccessControlMethodWhenProjectTeamEqualsNil(t *testing.T) {
	var attrs = storage.ACLRule{
		ProjectTeam: nil,
	}

	objectAccessControl := convertACLRuleToObjectAccessControl(attrs)

	assert.Nil(t, objectAccessControl.ProjectTeam)
}

func TestObjectAttrsToBucketObjectMethod(t *testing.T) {
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
		Finalized:               timeAttr,
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

	assert.Equal(t, attrs.Name, object.Name)
	assert.Equal(t, attrs.ContentType, object.ContentType)
	assert.Equal(t, attrs.ContentLanguage, object.ContentLanguage)
	assert.Equal(t, attrs.CacheControl, object.CacheControl)
	assert.Equal(t, attrs.Owner, object.Owner)
	assert.Equal(t, uint64(attrs.Size), object.Size)
	assert.Equal(t, attrs.ContentEncoding, object.ContentEncoding)
	assert.Equal(t, len(&md5Expected), len(object.MD5))
	assert.Equal(t, cap(&md5Expected), cap(object.MD5))
	assert.Equal(t, attrs.CRC32C, *object.CRC32C)
	assert.Equal(t, attrs.MediaLink, object.MediaLink)
	assert.Equal(t, attrs.Metadata, object.Metadata)
	assert.Equal(t, attrs.Generation, object.Generation)
	assert.Equal(t, attrs.Metageneration, object.MetaGeneration)
	assert.Equal(t, attrs.StorageClass, object.StorageClass)
	assert.Equal(t, attrs.Updated.String(), object.Updated.String())
	assert.Equal(t, attrs.Finalized.String(), object.Finalized.String())
	assert.Equal(t, attrs.Deleted.String(), object.Deleted.String())
	assert.Equal(t, attrs.ContentDisposition, object.ContentDisposition)
	assert.Equal(t, customeTimeExpected, object.CustomTime)
	assert.Equal(t, attrs.EventBasedHold, object.EventBasedHold)
	assert.Equal(t, acl, object.Acl)
	assert.Equal(t, attrs.ComponentCount, object.ComponentCount)
}

func TestConvertObjectAccessControlToACLRuleMethod(t *testing.T) {
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

	assert.Equal(t, storage.ACLEntity(objectAccessControl.Entity), aclRule.Entity)
	assert.Equal(t, objectAccessControl.EntityId, aclRule.EntityID)
	assert.Equal(t, storage.ACLRole(objectAccessControl.Role), aclRule.Role)
	assert.Equal(t, objectAccessControl.Domain, aclRule.Domain)
	assert.Equal(t, objectAccessControl.Email, aclRule.Email)
	assert.Equal(t, objectAccessControl.ProjectTeam.ProjectNumber, aclRule.ProjectTeam.ProjectNumber)
	assert.Equal(t, objectAccessControl.ProjectTeam.Team, aclRule.ProjectTeam.Team)
}

func TestConvertObjectAccessControlToACLRuleMethodWhenProjectTeamEqualsNil(t *testing.T) {
	objectAccessControl := &storagev1.ObjectAccessControl{
		ProjectTeam: nil,
	}

	aclRule := convertObjectAccessControlToACLRule(objectAccessControl)

	assert.Nil(t, aclRule.ProjectTeam)
}

func TestSetAttrsInWriterMethod(t *testing.T) {
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

	assert.Equal(t, createObjectRequest.Name, writer.Name)
	assert.Equal(t, createObjectRequest.ContentType, writer.ContentType)
	assert.Equal(t, createObjectRequest.ContentLanguage, writer.ContentLanguage)
	assert.Equal(t, createObjectRequest.ContentEncoding, writer.ContentEncoding)
	assert.Equal(t, createObjectRequest.CacheControl, writer.CacheControl)
	assert.Equal(t, createObjectRequest.Metadata, writer.Metadata)
	assert.Equal(t, createObjectRequest.ContentDisposition, writer.ContentDisposition)
	parsedTime, _ := time.Parse(time.RFC3339, createObjectRequest.CustomTime)
	assert.True(t, parsedTime.Equal(writer.CustomTime))
	assert.Equal(t, createObjectRequest.EventBasedHold, writer.EventBasedHold)
	assert.Equal(t, createObjectRequest.StorageClass, writer.StorageClass)
	assert.Equal(t, *createObjectRequest.CRC32C, writer.CRC32C)
	assert.True(t, writer.SendCRC32C)
	assert.Equal(t, string(createObjectRequest.MD5[:]), string(writer.MD5[:]))
}

func Test_ConvertObjToMinObject_WithNilObject(t *testing.T) {
	var gcsObject *gcs.Object

	gcsMinObject := ConvertObjToMinObject(gcsObject)

	assert.Nil(t, gcsMinObject)
}

func Test_ConvertObjToMinObject_WithValidObject(t *testing.T) {
	name := "test"
	size := uint64(36)
	generation := int64(444)
	metaGeneration := int64(555)
	currentTime := time.Now()
	contentEncode := "test_encoding"
	metadata := map[string]string{"test_key": "test_value"}
	var crc32C uint32 = 1234
	gcsObject := gcs.Object{
		Name:            name,
		Size:            size,
		Generation:      generation,
		MetaGeneration:  metaGeneration,
		Updated:         currentTime,
		Finalized:       currentTime,
		Metadata:        metadata,
		ContentEncoding: contentEncode,
		CRC32C:          &crc32C,
	}

	gcsMinObject := ConvertObjToMinObject(&gcsObject)

	require.NotNil(t, gcsMinObject)
	assert.Equal(t, name, gcsMinObject.Name)
	assert.Equal(t, size, gcsMinObject.Size)
	assert.Equal(t, generation, gcsMinObject.Generation)
	assert.Equal(t, metaGeneration, gcsMinObject.MetaGeneration)
	assert.True(t, currentTime.Equal(gcsMinObject.UpdatedTime()))
	assert.True(t, currentTime.Equal(gcsMinObject.FinalizedTime()))
	assert.Equal(t, contentEncode, gcsMinObject.ContentEncoding)
	assert.Equal(t, metadata, gcsMinObject.Metadata)
	assert.Equal(t, crc32C, *gcsMinObject.CRC32C)
}

func Test_ConvertObjToExtendedObjectAttributes_WithNilObject(t *testing.T) {
	var gcsObject *gcs.Object

	extendedObjAttr := ConvertObjToExtendedObjectAttributes(gcsObject)

	assert.Nil(t, extendedObjAttr)
}

func Test_ConvertObjToExtendedObjectAttributes_WithValidObject(t *testing.T) {
	var attrMd5 *[16]byte
	timeAttr := time.Now()
	gcsObject := gcs.Object{
		ContentType:        "ContentType",
		ContentLanguage:    "ContentLanguage",
		CacheControl:       "CacheControl",
		Owner:              "Owner",
		MD5:                attrMd5,
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

	require.NotNil(t, extendedObjAttr)
	assert.Equal(t, gcsObject.ContentType, extendedObjAttr.ContentType)
	assert.Equal(t, gcsObject.ContentLanguage, extendedObjAttr.ContentLanguage)
	assert.Equal(t, gcsObject.CacheControl, extendedObjAttr.CacheControl)
	assert.Equal(t, gcsObject.Owner, extendedObjAttr.Owner)
	assert.Equal(t, gcsObject.MD5, extendedObjAttr.MD5)
	assert.Equal(t, gcsObject.MediaLink, extendedObjAttr.MediaLink)
	assert.Equal(t, gcsObject.StorageClass, extendedObjAttr.StorageClass)
	assert.Equal(t, 0, gcsObject.Deleted.Compare(extendedObjAttr.Deleted))
	assert.Equal(t, gcsObject.ComponentCount, extendedObjAttr.ComponentCount)
	assert.Equal(t, gcsObject.ContentDisposition, extendedObjAttr.ContentDisposition)
	assert.Equal(t, gcsObject.CustomTime, extendedObjAttr.CustomTime)
	assert.Equal(t, gcsObject.EventBasedHold, extendedObjAttr.EventBasedHold)
	assert.Equal(t, gcsObject.Acl, extendedObjAttr.Acl)
}

func Test_ConvertObjToExtendedObjectAttributes_WithNilMinObjectAndNilAttributes(t *testing.T) {
	var minObject *gcs.MinObject
	var extendedObjectAttr *gcs.ExtendedObjectAttributes

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	assert.Nil(t, object)
}

func Test_ConvertObjToExtendedObjectAttributes_WithNilMinObjectAndNonNilAttributes(t *testing.T) {
	var minObject *gcs.MinObject
	extendedObjectAttr := &gcs.ExtendedObjectAttributes{
		ContentType: "ContentType",
	}

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	assert.Nil(t, object)
}

func Test_ConvertObjToExtendedObjectAttributes_WithNonNilMinObjectAndNilAttributes(t *testing.T) {
	name := "test"
	minObject := &gcs.MinObject{
		Name: name,
	}
	var extendedObjectAttr *gcs.ExtendedObjectAttributes

	object := ConvertMinObjectAndExtendedObjectAttributesToObject(minObject, extendedObjectAttr)

	assert.Nil(t, object)
}

func Test_ConvertObjToExtendedObjectAttributes_WithNonNilMinObjectAndNonNilAttributes(t *testing.T) {
	var attrMd5 *[16]byte
	timeAttr := time.Now()
	minObject := &gcs.MinObject{
		Name:            "test",
		Size:            uint64(36),
		Generation:      int64(444),
		MetaGeneration:  int64(555),
		Updated:         gcs.TimeToNS(timeAttr),
		Finalized:       gcs.TimeToNS(timeAttr),
		Metadata:        map[string]string{"test_key": "test_value"},
		ContentEncoding: "test_encoding",
	}
	extendedObjAttr := &gcs.ExtendedObjectAttributes{
		ContentType:        "ContentType",
		ContentLanguage:    "ContentLanguage",
		CacheControl:       "CacheControl",
		Owner:              "Owner",
		MD5:                attrMd5,
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

	require.NotNil(t, gcsObject)
	assert.Equal(t, minObject.Name, gcsObject.Name)
	assert.Equal(t, minObject.Size, gcsObject.Size)
	assert.Equal(t, minObject.Generation, gcsObject.Generation)
	assert.Equal(t, minObject.MetaGeneration, gcsObject.MetaGeneration)
	assert.Equal(t, 0, gcsObject.Updated.Compare(minObject.UpdatedTime()))
	assert.Equal(t, 0, gcsObject.Finalized.Compare(minObject.FinalizedTime()))
	assert.Equal(t, minObject.Metadata, gcsObject.Metadata)
	assert.Equal(t, minObject.ContentEncoding, gcsObject.ContentEncoding)
	assert.Equal(t, extendedObjAttr.ContentType, gcsObject.ContentType)
	assert.Equal(t, extendedObjAttr.ContentLanguage, gcsObject.ContentLanguage)
	assert.Equal(t, extendedObjAttr.CacheControl, gcsObject.CacheControl)
	assert.Equal(t, extendedObjAttr.Owner, gcsObject.Owner)
	assert.Equal(t, extendedObjAttr.MD5, gcsObject.MD5)
	assert.Equal(t, extendedObjAttr.MediaLink, gcsObject.MediaLink)
	assert.Equal(t, extendedObjAttr.StorageClass, gcsObject.StorageClass)
	assert.Equal(t, 0, gcsObject.Deleted.Compare(extendedObjAttr.Deleted))
	assert.Equal(t, extendedObjAttr.ComponentCount, gcsObject.ComponentCount)
	assert.Equal(t, extendedObjAttr.ContentDisposition, gcsObject.ContentDisposition)
	assert.Equal(t, extendedObjAttr.CustomTime, gcsObject.CustomTime)
	assert.Equal(t, extendedObjAttr.EventBasedHold, gcsObject.EventBasedHold)
	assert.Equal(t, extendedObjAttr.Acl, gcsObject.Acl)
}

func Test_ConvertMinObjectToObject_WithNilMinObject(t *testing.T) {
	var minObject *gcs.MinObject

	object := ConvertMinObjectToObject(minObject)

	assert.Nil(t, object)
}

func Test_ConvertMinObjectToObject_WithNonNilMinObject(t *testing.T) {
	var attrMd5 *[16]byte
	var crc32C uint32 = 1234
	timeAttr := time.Now()
	minObject := &gcs.MinObject{
		Name:            "test",
		Size:            uint64(36),
		Generation:      int64(444),
		MetaGeneration:  int64(555),
		Updated:         gcs.TimeToNS(timeAttr),
		Finalized:       gcs.TimeToNS(timeAttr),
		Metadata:        map[string]string{"test_key": "test_value"},
		ContentEncoding: "test_encoding",
		CRC32C:          &crc32C,
	}

	gcsObject := ConvertMinObjectToObject(minObject)

	require.NotNil(t, gcsObject)
	assert.Equal(t, minObject.Name, gcsObject.Name)
	assert.Equal(t, minObject.Size, gcsObject.Size)
	assert.Equal(t, minObject.Generation, gcsObject.Generation)
	assert.Equal(t, minObject.MetaGeneration, gcsObject.MetaGeneration)
	assert.Equal(t, 0, gcsObject.Updated.Compare(minObject.UpdatedTime()))
	assert.Equal(t, 0, gcsObject.Finalized.Compare(minObject.FinalizedTime()))
	assert.Equal(t, minObject.Metadata, gcsObject.Metadata)
	assert.Equal(t, minObject.ContentEncoding, gcsObject.ContentEncoding)
	assert.Equal(t, "", gcsObject.ContentType)
	assert.Equal(t, "", gcsObject.ContentLanguage)
	assert.Equal(t, "", gcsObject.CacheControl)
	assert.Equal(t, "", gcsObject.Owner)
	assert.Equal(t, attrMd5, gcsObject.MD5)
	assert.Equal(t, crc32C, *gcsObject.CRC32C)
	assert.Equal(t, "", gcsObject.MediaLink)
	assert.Equal(t, "", gcsObject.StorageClass)
	assert.Equal(t, 0, gcsObject.Deleted.Compare(time.Time{}))
	assert.Equal(t, int64(0), gcsObject.ComponentCount)
	assert.Equal(t, "", gcsObject.ContentDisposition)
	assert.Equal(t, "", gcsObject.CustomTime)
	assert.Equal(t, false, gcsObject.EventBasedHold)
	assert.Equal(t, []*storagev1.ObjectAccessControl(nil), gcsObject.Acl)
}
