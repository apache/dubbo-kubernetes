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

package appflag

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app/applog"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app/appname"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app/appverbose"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/verbose"
	"go.uber.org/zap"
)

type container struct {
	app.Container
	nameContainer    appname.Container
	logContainer     applog.Container
	verboseContainer appverbose.Container
}

func newContainer(
	baseContainer app.Container,
	appName string,
	logger *zap.Logger,
	verbosePrinter verbose.Printer,
) (*container, error) {
	nameContainer, err := appname.NewContainer(baseContainer, appName)
	if err != nil {
		return nil, err
	}
	return &container{
		Container:        baseContainer,
		nameContainer:    nameContainer,
		logContainer:     applog.NewContainer(logger),
		verboseContainer: appverbose.NewContainer(verbosePrinter),
	}, nil
}

func (c *container) AppName() string {
	return c.nameContainer.AppName()
}

func (c *container) ConfigDirPath() string {
	return c.nameContainer.ConfigDirPath()
}

func (c *container) CacheDirPath() string {
	return c.nameContainer.CacheDirPath()
}

func (c *container) DataDirPath() string {
	return c.nameContainer.DataDirPath()
}

func (c *container) Port() (uint16, error) {
	return c.nameContainer.Port()
}

func (c *container) Logger() *zap.Logger {
	return c.logContainer.Logger()
}

func (c *container) VerbosePrinter() verbose.Printer {
	return c.verboseContainer.VerbosePrinter()
}
