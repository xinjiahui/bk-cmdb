/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package x19_05_16_01

import (
	"context"
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

func addInnerCategory(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	innerCategories := []struct {
		Name       string `field:"name" json:"name,omitempty" bson:"name"`
		ParentName string `field:"bk_parent_id" json:"bk_parent_id,omitempty" bson:"bk_parent_id"`
	}{
		{
			Name:       "数据库",
			ParentName: "",
		}, {
			Name:       "mysql",
			ParentName: "数据库",
		}, {
			Name:       "oracle",
			ParentName: "数据库",
		}, {
			Name:       "redis",
			ParentName: "数据库",
		}, {
			Name:       "web中间件",
			ParentName: "",
		}, {
			Name:       "apache",
			ParentName: "web中间件",
		}, {
			Name:       "nginx",
			ParentName: "web中间件",
		},
	}

	exist := false
	categoryIDMap := map[string]int64{}
	for _, category := range innerCategories {
		parentID := int64(0)
		if len(category.ParentName) > 0 {
			parentID, exist = categoryIDMap[category.ParentName]
			if exist == false {
				return fmt.Errorf("parent [%s] not found", category.ParentName)
			}
		}
		categoryID, err := getOrCreateCategory(ctx, db, category.Name, parentID)
		if err != nil {
			return fmt.Errorf("get or create category failed, err: %+v", err)
		}
		categoryIDMap[category.Name] = categoryID
	}
	return nil
}

func getOrCreateCategory(ctx context.Context, db dal.RDB, name string, parentID int64) (int64, error) {
	category := metadata.ServiceCategory{}
	filter := map[string]interface{}{
		common.MetadataLabelBiz: mapstr.MapStr{common.BKDBExists: false},
		common.BKFieldName:      name,
		common.BKParentIDField:  parentID,
	}
	err := db.Table(common.BKTableNameServiceCategory).Find(filter).One(ctx, &category)
	if db.IsNotFoundError(err) {
		categoryID, err := db.NextSequence(ctx, common.BKTableNameServiceCategory)
		if err != nil {
			return 0, fmt.Errorf("generate category id failed, err: %+v", err)
		}

		rootID := int64(0)
		if parentID != 0 {
			parentCategory := &metadata.ServiceCategory{}
			parentFilter := map[string]interface{}{
				common.BKFieldID: parentID,
			}
			if err := db.Table(common.BKTableNameServiceCategory).Find(parentFilter).One(ctx, parentCategory); err != nil {
				return 0, fmt.Errorf("get parent category: %d failed, err: %+v", parentID, err)
			}
			rootID = parentCategory.RootID
		}

		category = metadata.ServiceCategory{
			ID:              int64(categoryID),
			Name:            name,
			RootID:          rootID,
			ParentID:        parentID,
			SupplierAccount: "0",
			IsBuiltIn:       true,
		}
		err = db.Table(common.BKTableNameServiceCategory).Insert(ctx, category)
		if err != nil {
			return 0, fmt.Errorf("create service category failed, category: %+v, err: %+v", category, err)
		}
	}
	return category.ID, nil
}
