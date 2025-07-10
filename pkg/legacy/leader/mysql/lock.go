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
	"context"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const TIMEOUT int64 = 10 * 60

type MysqlLock struct {
	id         string
	expireTime int64
	db         *gorm.DB
}

type DistributedLock struct {
	Id         string `gorm:"primary_key"`
	ExpireTime int64
}

func NewLock(id string, db *gorm.DB) *MysqlLock {
	return &MysqlLock{
		id:         id,
		expireTime: time.Now().Unix() + TIMEOUT,
		db:         db,
	}
}

func (lock *MysqlLock) TryLock() (bool, error) {
	// clean timeout lock
	lock.unLock()
	newLock := DistributedLock{
		Id:         lock.id,
		ExpireTime: lock.expireTime,
	}
	// insert lock
	err := lock.db.Table("distributed_lock").Create(&newLock).Error
	if err != nil {
		// If the primary key is existed, return false, as this is an expected error
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (lock *MysqlLock) KeepLock(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			oldLock := DistributedLock{
				Id: lock.id,
			}
			lock.db.Table("distributed_lock").Model(&oldLock).Update("expire_time", time.Now().Unix()+TIMEOUT)
			timer := time.NewTimer(time.Second * 10)
			<-timer.C
		}
	}
}

func (lock *MysqlLock) unLock() {
	now := time.Now().Unix()
	lock.db.Table("distributed_lock").Where("id = ? AND expire_time < ?", lock.id, now).Delete(DistributedLock{})
}
