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
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func RenderJSONOrError(c *gin.Context, value interface{}, err error) {
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	} else {
		c.JSON(200, value)
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

	routerGroup.GET("/healthcheck", func(c *gin.Context) {
		c.String(200, "ok")
	})

	routerGroup.GET("/machines", func(c *gin.Context) {
		machines, err := (*cl.Client).Machines()

		RenderJSONOrError(c, machines, err)
	})

	routerGroup.GET("/units/:name", func(c *gin.Context) {
		unit, err := (*cl.Client).Unit(normalizeName(c.Params.ByName("name")))

		RenderJSONOrError(c, unit, err)
	})

	routerGroup.GET("/units", func(c *gin.Context) {
		units, err := (*cl.Client).Units()

		RenderJSONOrError(c, units, err)
	})

	routerGroup.PUT("/deploy/:name", func(c *gin.Context) {
		log.Printf("Deploy asked for %s", normalizeName(c.Params.ByName("name")))

		log.Printf(
			"Deploy wil be done on %s",
			strings.Join(
				cl.FindMatchingUnits(normalizeName(c.Params.ByName("name"))),
				"",
			),
		)

		go func() {
			for _, unit := range cl.FindMatchingUnits(normalizeName(c.Params.ByName("name"))) {
				log.Printf("prepare restart on %s", unit)
				cl.RestartUnit(unit)
			}
		}()
		c.JSON(200, "Deployment asked")
	})

	routerGroup.PUT("/rebalance/:name", func(c *gin.Context) {
		log.Printf("Rebalance asked for %s", normalizeName(c.Params.ByName("name")))

		log.Printf(
			"Rebalance wil be done on %s",
			strings.Join(
				cl.FindMatchingUnits(normalizeName(c.Params.ByName("name"))),
				"",
			),
		)

		go func() {
			for _, unit := range cl.FindMatchingUnits(normalizeName(c.Params.ByName("name"))) {
				log.Printf("prepare restart on %s", unit)
				cl.RebalanceUnit(unit)
			}
		}()

		c.JSON(200, "Rebalancing asked")
	})

	routerEngine.Run(":8080")
}
