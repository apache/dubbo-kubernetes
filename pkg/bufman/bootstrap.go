/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bufman

import (
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"

	_ "github.com/apache/dubbo-kubernetes/pkg/admin/cache/registry/kube"
	_ "github.com/apache/dubbo-kubernetes/pkg/admin/cache/registry/universal"
)

func InitConfig(rt core_runtime.Runtime) error {
	config.Properties = &rt.Config().Bufman

	return nil
}

func RegisterDatabase(rt core_runtime.Runtime) error {
	dsn := rt.Config().Bufman.MySQL.MysqlDsn
	var db *gorm.DB
	var err error
	if dsn == "" {
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	} else {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		return err
	} else {
		config.DataBase = db
		// init table
		initErr := db.AutoMigrate(
			&model.Repository{},
			&model.Commit{},
			&model.Tag{},
			&model.User{},
			&model.Token{},
			&model.CommitFile{},
			&model.FileBlob{},
		)
		if initErr != nil {
			return initErr
		}
	}

	rawDB, err := config.DataBase.DB()
	if err != nil {
		return err
	}

	rawDB.SetMaxOpenConns(config.Properties.MySQL.MaxOpenConnections)
	rawDB.SetMaxIdleConns(config.Properties.MySQL.MaxIdleConnections)
	rawDB.SetConnMaxLifetime(config.Properties.MySQL.MaxLifeTime)
	rawDB.SetConnMaxIdleTime(config.Properties.MySQL.MaxIdleTime)

	return nil
}
