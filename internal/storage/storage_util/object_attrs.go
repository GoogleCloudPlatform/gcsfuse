// Please don't review this file, this will be synced with Tulsi's changes.

package storage_util

import (
	"crypto/md5"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	storagev1 "google.golang.org/api/storage/v1"
)

func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *gcs.Object {
	// Converting []ACLRule returned by the Go Client into []*storagev1.ObjectAccessControl which complies with GCSFuse type.
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

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var MD5 [md5.Size]byte
	copy(MD5[:], attrs.MD5)

	// Setting the parameters in Object and doing conversions as necessary.
	return &gcs.Object{
		Name:            attrs.Name,
		ContentType:     attrs.ContentType,
		ContentLanguage: attrs.ContentLanguage,
		CacheControl:    attrs.CacheControl,
		Owner:           attrs.Owner,
		Size:            uint64(attrs.Size),
		ContentEncoding: attrs.ContentEncoding,
		MD5:             &MD5,
		CRC32C:          &attrs.CRC32C,
		MediaLink:       attrs.MediaLink,
		Metadata:        attrs.Metadata,
		Generation:      attrs.Generation,
		MetaGeneration:  attrs.Metageneration,
		StorageClass:    attrs.StorageClass,
		Deleted:         attrs.Deleted,
		Updated:         attrs.Updated,
		//ComponentCount: , (Field not found in attrs returned by Go Client.)
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                Acl,
	}
}

// Function for setting attributes to the Writer. These attributes will be assigned to the newly created object / already existing object.
func SetAttrs(wc *storage.Writer, req *gcs.CreateObjectRequest) *storage.Writer {
	wc.Name = req.Name
	wc.ContentType = req.ContentType
	wc.ContentLanguage = req.ContentLanguage
	wc.ContentEncoding = req.ContentLanguage
	wc.CacheControl = req.CacheControl
	wc.Metadata = req.Metadata
	wc.ContentDisposition = req.ContentDisposition
	wc.CustomTime, _ = time.Parse(time.RFC3339, req.CustomTime)
	wc.EventBasedHold = req.EventBasedHold
	wc.StorageClass = req.StorageClass

	// Converting []*storagev1.ObjectAccessControl to []ACLRule as expected by the Go Client Writer.
	var Acl []storage.ACLRule
	for _, element := range req.Acl {
		currACL := storage.ACLRule{
			Entity:   storage.ACLEntity(element.Entity),
			EntityID: element.EntityId,
			Role:     storage.ACLRole(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storage.ProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		Acl = append(Acl, currACL)
	}
	wc.ACL = Acl

	if req.CRC32C != nil {
		wc.CRC32C = *req.CRC32C
		wc.SendCRC32C = true // Explicitly need to send CRC32C token in Writer in order to send the checksum.
	}

	if req.MD5 != nil {
		wc.MD5 = (*req.MD5)[:]
	}

	return wc
}
