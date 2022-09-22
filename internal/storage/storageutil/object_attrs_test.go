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

func TestObjectAttrs(t *testing.T) { RunTests(t) }

type objectAttrsTest struct {
}

func init() { RegisterTestSuite(&objectAttrsTest{}) }

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
	}
	customeTimeExpected := string(attrs.CustomTime.Format(time.RFC3339))

	var md5Expected [md5.Size]byte
	copy(md5Expected[:], attrs.MD5)

	var acl []*storagev1.ObjectAccessControl
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
		acl = append(acl, currACL)
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
	ExpectEq(object.CRC32C, &attrs.CRC32C)
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
}
