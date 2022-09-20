package storageutil

import (
	"crypto/md5"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	. "github.com/jacobsa/ogletest"
	storagev1 "google.golang.org/api/storage/v1"
)

const TestBucketName string = "gcsfuse-default-bucket"
const TestObjectName string = "gcsfuse/default.txt"

func TestObjectAttrsToBucketObject(t *testing.T) { RunTests(t) }

type objectAttrsToBucketObject struct {
}

func init() { RegisterTestSuite(&objectAttrsToBucketObject{}) }

func (t objectAttrsToBucketObject) TestObjectAttrsToBucketObjectMethod() {
	var attrMd5 []byte
	Time := time.Now()
	attrs := storage.ObjectAttrs{
		Bucket:                  TestBucketName,
		Name:                    TestObjectName,
		ContentType:             "",
		ContentLanguage:         "",
		CacheControl:            "",
		EventBasedHold:          false,
		TemporaryHold:           false,
		RetentionExpirationTime: Time,
		ACL:                     nil,
		PredefinedACL:           "",
		Owner:                   "",
		Size:                    16,
		ContentEncoding:         "",
		ContentDisposition:      "",
		MD5:                     attrMd5,
		CRC32C:                  0,
		MediaLink:               "",
		Metadata:                nil,
		Generation:              780,
		Metageneration:          0,
		StorageClass:            "",
		Created:                 Time,
		Deleted:                 Time,
		Updated:                 Time,
		CustomerKeySHA256:       "",
		KMSKeyName:              "",
		Prefix:                  "",
		Etag:                    "",
		CustomTime:              Time,
	}
	CustomeTimeExpected := string(attrs.CustomTime.Format(time.RFC3339))

	var MD5Expected [md5.Size]byte
	copy(MD5Expected[:], attrs.MD5)

	var Acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		currACL := &storagev1.ObjectAccessControl{
			Entity:   string(element.Entity),
			EntityId: element.EntityID,
			Role:     string(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		Acl = append(Acl, currACL)
	}

	object := ObjectAttrsToBucketObject(&attrs)

	ExpectEq(object.Name, attrs.Name)
	ExpectEq(object.ContentType, attrs.ContentType)
	ExpectEq(object.ContentLanguage, attrs.ContentLanguage)
	ExpectEq(object.CacheControl, attrs.CacheControl)
	ExpectEq(object.Owner, attrs.Owner)
	ExpectEq(object.Size, attrs.Size)
	ExpectEq(object.ContentEncoding, attrs.ContentEncoding)
	ExpectEq(len(object.MD5), len(&MD5Expected))
	ExpectEq(cap(object.MD5), cap(&MD5Expected))
	ExpectEq(object.CRC32C, &attrs.CRC32C)
	ExpectEq(object.MediaLink, attrs.MediaLink)
	ExpectEq(object.Metadata, attrs.Metadata)
	ExpectEq(object.Generation, attrs.Generation)
	ExpectEq(object.MetaGeneration, attrs.Metageneration)
	ExpectEq(object.StorageClass, attrs.StorageClass)
	ExpectEq(object.Updated.Day(), attrs.Updated.Day())
	ExpectEq(object.Updated.Month(), attrs.Updated.Month())
	ExpectEq(object.Updated.Year(), attrs.Updated.Year())
	ExpectEq(object.Updated.Hour(), attrs.Updated.Hour())
	ExpectEq(object.Updated.Minute(), attrs.Updated.Minute())
	ExpectEq(object.Updated.Second(), attrs.Updated.Second())
	ExpectEq(object.Updated.UnixMicro(), attrs.Updated.UnixMicro())
	ExpectEq(object.Deleted.Day(), attrs.Deleted.Day())
	ExpectEq(object.Deleted.Month(), attrs.Deleted.Month())
	ExpectEq(object.Deleted.Year(), attrs.Deleted.Year())
	ExpectEq(object.Deleted.Hour(), attrs.Deleted.Hour())
	ExpectEq(object.Deleted.Minute(), attrs.Deleted.Minute())
	ExpectEq(object.Deleted.Second(), attrs.Deleted.Second())
	ExpectEq(object.Deleted.UnixMicro(), attrs.Deleted.UnixMicro())
	ExpectEq(object.ContentDisposition, attrs.ContentDisposition)
	ExpectEq(object.CustomTime, CustomeTimeExpected)
	ExpectEq(object.EventBasedHold, attrs.EventBasedHold)
	ExpectEq(object.Acl, Acl)
}
