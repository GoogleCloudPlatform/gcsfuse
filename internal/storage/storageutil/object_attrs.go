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
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	storagev1 "google.golang.org/api/storage/v1"
)

func convertObjectAccessControlToACLRule(obj *storagev1.ObjectAccessControl) storage.ACLRule {
	aclObj := storage.ACLRule{
		Entity:   storage.ACLEntity(obj.Entity),
		EntityID: obj.EntityId,
		Role:     storage.ACLRole(obj.Role),
		Domain:   obj.Domain,
		Email:    obj.Email,
	}

	if obj.ProjectTeam != nil {
		aclObj.ProjectTeam = &storage.ProjectTeam{
			ProjectNumber: obj.ProjectTeam.ProjectNumber,
			Team:          obj.ProjectTeam.Team,
		}
	}

	return aclObj
}

func convertACLRuleToObjectAccessControl(element storage.ACLRule) *storagev1.ObjectAccessControl {
	obj := &storagev1.ObjectAccessControl{
		Entity:   string(element.Entity),
		EntityId: element.EntityID,
		Role:     string(element.Role),
		Domain:   element.Domain,
		Email:    element.Email,
	}

	if element.ProjectTeam != nil {
		obj.ProjectTeam = &storagev1.ObjectAccessControlProjectTeam{
			ProjectNumber: element.ProjectTeam.ProjectNumber,
			Team:          element.ProjectTeam.Team,
		}
	}

	return obj
}

func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *gcs.Object {
	// gcs.Object accepts []*storagev1.ObjectAccessControl instead of []ACLRule.
	var acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		acl = append(acl, convertACLRuleToObjectAccessControl(element))
	}

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var md5 [md5.Size]byte
	copy(md5[:], attrs.MD5)

	// Making a local copy of crc to avoid keeping a reference to attrs instance.
	crc := attrs.CRC32C

	// Setting the parameters in Object and doing conversions as necessary.
	return &gcs.Object{
		Name:               attrs.Name,
		ContentType:        attrs.ContentType,
		ContentLanguage:    attrs.ContentLanguage,
		CacheControl:       attrs.CacheControl,
		Owner:              attrs.Owner,
		Size:               uint64(attrs.Size),
		ContentEncoding:    attrs.ContentEncoding,
		MD5:                &md5,
		CRC32C:             &crc,
		MediaLink:          attrs.MediaLink,
		Metadata:           attrs.Metadata,
		Generation:         attrs.Generation,
		MetaGeneration:     attrs.Metageneration,
		StorageClass:       attrs.StorageClass,
		Deleted:            attrs.Deleted,
		Updated:            attrs.Updated,
		ComponentCount:     attrs.ComponentCount,
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                acl,
	}
}

// SetAttrsInWriter - for setting object-attributes filed in storage.Writer object.
// These attributes will be assigned to the newly created or old object.
func SetAttrsInWriter(wc *storage.Writer, req *gcs.CreateObjectRequest) *storage.Writer {
	wc.Name = req.Name
	wc.ContentType = req.ContentType
	wc.ContentLanguage = req.ContentLanguage
	wc.ContentEncoding = req.ContentEncoding
	wc.CacheControl = req.CacheControl
	wc.Metadata = req.Metadata
	wc.ContentDisposition = req.ContentDisposition
	wc.CustomTime, _ = time.Parse(time.RFC3339, req.CustomTime)
	wc.EventBasedHold = req.EventBasedHold
	wc.StorageClass = req.StorageClass

	// Converting []*storagev1.ObjectAccessControl to []ACLRule for writer object.
	var aclRules []storage.ACLRule
	for _, element := range req.Acl {
		aclRules = append(aclRules, convertObjectAccessControlToACLRule(element))
	}
	wc.ACL = aclRules

	if req.CRC32C != nil {
		wc.CRC32C = *req.CRC32C
		wc.SendCRC32C = true
	}

	if req.MD5 != nil {
		wc.MD5 = (*req.MD5)[:]
	}

	return wc
}

func ConvertObjToMinObject(o *gcs.Object) *gcs.MinObject {
	if o == nil {
		return nil
	}

	return &gcs.MinObject{
		Name:            o.Name,
		Size:            o.Size,
		Generation:      o.Generation,
		MetaGeneration:  o.MetaGeneration,
		Updated:         o.Updated,
		Metadata:        o.Metadata,
		ContentEncoding: o.ContentEncoding,
		CRC32C:          o.CRC32C,
	}
}

func ConvertObjToExtendedObjectAttributes(o *gcs.Object) *gcs.ExtendedObjectAttributes {
	if o == nil {
		return nil
	}

	return &gcs.ExtendedObjectAttributes{
		ContentType:        o.ContentType,
		ContentLanguage:    o.ContentLanguage,
		CacheControl:       o.CacheControl,
		Owner:              o.Owner,
		MD5:                o.MD5,
		MediaLink:          o.MediaLink,
		StorageClass:       o.StorageClass,
		Deleted:            o.Deleted,
		ComponentCount:     o.ComponentCount,
		ContentDisposition: o.ContentDisposition,
		CustomTime:         o.CustomTime,
		EventBasedHold:     o.EventBasedHold,
		Acl:                o.Acl,
	}
}

func ConvertMinObjectAndExtendedObjectAttributesToObject(m *gcs.MinObject,
	e *gcs.ExtendedObjectAttributes) *gcs.Object {
	if m == nil || e == nil {
		return nil
	}

	return &gcs.Object{
		Name:               m.Name,
		Size:               m.Size,
		Generation:         m.Generation,
		MetaGeneration:     m.MetaGeneration,
		Updated:            m.Updated,
		Metadata:           m.Metadata,
		ContentEncoding:    m.ContentEncoding,
		ContentType:        e.ContentType,
		ContentLanguage:    e.ContentLanguage,
		CacheControl:       e.CacheControl,
		Owner:              e.Owner,
		MD5:                e.MD5,
		CRC32C:             m.CRC32C,
		MediaLink:          e.MediaLink,
		StorageClass:       e.StorageClass,
		Deleted:            e.Deleted,
		ComponentCount:     e.ComponentCount,
		ContentDisposition: e.ContentDisposition,
		CustomTime:         e.CustomTime,
		EventBasedHold:     e.EventBasedHold,
		Acl:                e.Acl,
	}
}

func ConvertMinObjectToObject(m *gcs.MinObject) *gcs.Object {
	if m == nil {
		return nil
	}

	return &gcs.Object{
		Name:            m.Name,
		Size:            m.Size,
		Generation:      m.Generation,
		MetaGeneration:  m.MetaGeneration,
		Updated:         m.Updated,
		Metadata:        m.Metadata,
		ContentEncoding: m.ContentEncoding,
		CRC32C:          m.CRC32C,
	}
}
