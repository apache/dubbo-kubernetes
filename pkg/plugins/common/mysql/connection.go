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

package mysql

import (
	"github.com/pkg/errors"

	mysql_driver "gorm.io/driver/mysql"

	sqlite_driver "gorm.io/driver/sqlite"

	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/resources/mysql"
	leader_mysql "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/mysql"
)

func ConnectToDb(cfg mysql.MysqlStoreConfig) (*gorm.DB, error) {
	dsn := cfg.MysqlDsn
	var db *gorm.DB
	var err error
	if dsn == "" {
		db, err = gorm.Open(sqlite_driver.Open(":memory:"), &gorm.Config{})
	} else {
		db, err = gorm.Open(mysql_driver.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		return nil, err
	}

	initErr := db.AutoMigrate(
		&leader_mysql.DistributedLock{},
	)
	if initErr != nil {
		return nil, initErr
	}
	rawDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// check connection to DB, Open() does not check it.
	if err := rawDB.Ping(); err != nil {
		return nil, errors.Wrap(err, "cannot connect to DB")
	}

	rawDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	rawDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	rawDB.SetConnMaxLifetime(cfg.MaxLifeTime)
	rawDB.SetConnMaxIdleTime(cfg.MaxIdleTime)

	return db, nil
}
