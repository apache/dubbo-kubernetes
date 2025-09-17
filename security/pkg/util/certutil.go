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

package util

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"time"
)

type CertUtil interface {
	GetWaitTime([]byte, time.Time) (time.Duration, error)
}

type CertUtilImpl struct {
	gracePeriodPercentage int
}

func NewCertUtil(gracePeriodPercentage int) CertUtilImpl {
	return CertUtilImpl{
		gracePeriodPercentage: gracePeriodPercentage,
	}
}

func (cu CertUtilImpl) GetWaitTime(certBytes []byte, now time.Time) (time.Duration, error) {
	cert, certErr := util.ParsePemEncodedCertificate(certBytes)
	if certErr != nil {
		return time.Duration(0), certErr
	}
	timeToExpire := cert.NotAfter.Sub(now)
	if timeToExpire < 0 {
		return time.Duration(0), fmt.Errorf("certificate already expired at %s, but now is %s",
			cert.NotAfter, now)
	}
	// Note: multiply time.Duration(int64) by an int (gracePeriodPercentage) will cause overflow (e.g.,
	// when duration is time.Hour * 90000). So float64 is used instead.
	gracePeriod := time.Duration(float64(cert.NotAfter.Sub(cert.NotBefore)) * (float64(cu.gracePeriodPercentage) / 100))
	// waitTime is the duration between now and the grace period starts.
	// It is the time until cert expiration minus the length of grace period.
	waitTime := timeToExpire - gracePeriod
	if waitTime < 0 {
		// We are within the grace period.
		return time.Duration(0), fmt.Errorf("got a certificate that should be renewed now")
	}
	return waitTime, nil
}
