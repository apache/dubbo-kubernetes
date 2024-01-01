// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() {
	dsn := config.Properties.MySQL.MysqlDsn
	var DB *gorm.DB
	var err error
	if dsn == "" {
		panic("MySQL DSN is empty")
	} else {
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	}

	if err != nil {
		panic(err)
	} else {
		config.DataBase = DB
		// init table
		initErr := DB.AutoMigrate(
			&Repository{},
			&Commit{},
			&Tag{},
			&User{},
			&Token{},
			&FileManifest{},
			&FileBlob{},
		)
		if initErr != nil {
			panic(initErr)
		}
	}

	db, err := config.DataBase.DB()
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(config.Properties.MySQL.MaxOpenConnections)
	db.SetMaxIdleConns(config.Properties.MySQL.MaxIdleConnections)
	db.SetConnMaxLifetime(config.Properties.MySQL.MaxLifeTime)
	db.SetConnMaxIdleTime(config.Properties.MySQL.MaxIdleTime)
}
