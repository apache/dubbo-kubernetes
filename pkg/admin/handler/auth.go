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

package handler

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Login(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.PostForm("user")
		password := c.PostForm("password")
		// verify username and password
		if user == rt.Config().Admin.Auth.User && password == rt.Config().Admin.Auth.Password {
			session := sessions.Default(c)
			session.Set("user", user)
			session.Options(sessions.Options{
				MaxAge: rt.Config().Admin.Auth.ExpirationTime,
				Path:   "/",
			})
			err := session.Save()
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(nil))
		} else {
			c.JSON(http.StatusUnauthorized, model.NewUnauthorizedResp())
		}
	}
}

func Logout(_ core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		err := session.Save()
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
