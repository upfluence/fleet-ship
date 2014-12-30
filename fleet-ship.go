/*
   Copyright 2014 Upfluence, Inc.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
  "strings"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/job"
	"github.com/gin-gonic/gin"
)

func NewFleetAPIClient(path string) (client.API, error) {
	ep, _ := url.Parse(path)
	dialUnix := ep.Scheme == "unix" || ep.Scheme == "file"
	dialFunc := net.Dial

	if dialUnix {
		ep.Host = "domain-sock"
		ep.Scheme = "http"

		sockPath := ep.Path

		ep.Path = ""

		dialFunc = func(string, string) (net.Conn, error) {
			return net.Dial("unix", sockPath)
		}
	}

	tr := &http.Transport{
		Dial: dialFunc,
	}

	cl := http.Client{Transport: tr}

	return client.NewHTTPClient(&cl, *ep)
}

func RenderJSONOrError(c *gin.Context, value interface{}, err error) {
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	} else {
		c.JSON(200, value)
	}
}

func waitUntilTargetStateReached(client client.API, name, state string) {
	sleep := 500 * time.Millisecond

	for {
		unit, err := client.Unit(name)

		if err == nil && unit.CurrentState == state {
			return
		}

		time.Sleep(sleep)
	}
}

func normalizeName(name string) string {
  if strings.HasSuffix(name, ".service") {
    return name
  } else {
    return name + ".service"
  }
}

func main() {
	endpoint := os.Getenv("FLEET_ENDPOINT")

	cl, _ := NewFleetAPIClient(endpoint)

	routerEngine := gin.Default()

	routerGroup := routerEngine.RouterGroup

	if endpoint == "" {
		endpoint = "unix://var/run/fleet.sock"
	}

	if os.Getenv("BASIC_USERNAME") != "" || os.Getenv("BASIC_PASSWORD") != "" {
		routerGroup = routerEngine.Group(
			"/",
			gin.BasicAuth(gin.Accounts{
				os.Getenv("BASIC_USERNAME"): os.Getenv("BASIC_PASSWORD"),
			}),
		)
	}

	routerGroup.GET("/machines", func(c *gin.Context) {
		machines, err := cl.Machines()

		RenderJSONOrError(c, machines, err)
	})

	routerGroup.GET("/units/:name", func(c *gin.Context) {
		unit, err := cl.Unit(normalizeName(c.Params.ByName("name")))

		RenderJSONOrError(c, unit, err)
	})

	routerGroup.GET("/units", func(c *gin.Context) {
		units, err := cl.Units()

		RenderJSONOrError(c, units, err)
	})

	routerGroup.PUT("/deploy/:name", func(c *gin.Context) {
		var err error

		name := normalizeName(c.Params.ByName("name"))

		err = cl.SetUnitTargetState(name, string(job.JobStateLoaded))

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}

		go func() {
			waitUntilTargetStateReached(cl, name, string(job.JobStateLoaded))

			cl.SetUnitTargetState(name, string(job.JobStateLaunched))
		}()

		c.JSON(200, "Deployment asked")
	})

	routerGroup.PUT("/rebalance/:name", func(c *gin.Context) {
		var err error

		name := normalizeName(c.Params.ByName("name"))

		err = cl.SetUnitTargetState(name, string(job.JobStateInactive))

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}

		go func() {
			waitUntilTargetStateReached(cl, name, string(job.JobStateInactive))

			cl.SetUnitTargetState(name, string(job.JobStateLaunched))
		}()

		c.JSON(200, "Rebalancing asked")
	})

	routerEngine.Run(":8080")
}
