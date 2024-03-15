/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package components

import (
	"fmt"
	"net"
	"strconv"
)

// FormatAddrArg formats the given addr and nodeId to a valid socket string.
// This function will return an empty string when the given addr is empty.
func FormatAddrArg(addr string, nodeId int) string {
	// return empty result if the address is not specified
	if len(addr) == 0 {
		return addr
	}

	// The "addr" is validated when set.
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.Atoi(port)

	return net.JoinHostPort(host, strconv.Itoa(portInt+nodeId))
}

// GenerateAddrArg pushes arg into args array, return the new args array.
func GenerateAddrArg(config string, addr string, nodeId int, args []string) []string {
	socketAddr := FormatAddrArg(addr, nodeId)

	// don't generate param if the socket address is empty
	if len(socketAddr) == 0 {
		return args
	}

	return append(args, fmt.Sprintf("%s=%s", config, socketAddr))
}
