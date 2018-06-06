package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/r0wbrt/riot/pkg/riotclient"
)

func main() {

	fmt.Println("rIOT Server Inspector")
	url := strings.Join(os.Args[1:], "")

	fmt.Printf("Connecting to rIOT server %s \r\n", url)

	rs, err := riotclient.Initialize(context.Background(), url)

	if err != nil {
		panic(err)
	}

	fmt.Println("Connection to rIOT Server succeeded")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("Basic Server Information")
	fmt.Println("===========")
	fmt.Println("")
	fmt.Println("")
	fmt.Printf("Server Name: %s\r\n", rs.Name)
	fmt.Println("\t\t---\t\t")
	fmt.Printf("Server GUID: %s\r\n", rs.GUID)
	fmt.Println("\t\t---\t\t")
	fmt.Printf("Server Description: %s\r\n", rs.Description)
	fmt.Println("")
	fmt.Println("")

	list, err := rs.GetResourceList(context.Background())
	if err != nil {
		panic(err)
	}

	if len(list) > 0 {
		fmt.Println("Server Resources Found")
		fmt.Println("===========")
		fmt.Println("")
		fmt.Println("")
		for i := 0; i < len(list); i++ {
			fmt.Printf("Resource -- %s --\r\n", list[i])

			resp, err := rs.GetResource(context.Background(), list[i])
			if err != nil {
				panic(err)
			}

			fmt.Printf("Resource Name: %s\r\n", resp.Name)
			fmt.Printf("Resource Description: %s\r\n", resp.Description)
			fmt.Printf("Resource Retention: %s\r\n", resp.RetentionPolicy)

			scS := []string{}
			for j := 0; j < len(resp.Schema); j++ {
				scS = append(scS, fmt.Sprintf("%s (%s,%s)", resp.Schema[j].Name, resp.Schema[j].MeasurmentUnit, resp.Schema[j].StorageUnit))
			}

			fmt.Printf("Resource Schema: %s\r\n", strings.Join(scS, ","))

			if i != len(list)-1 {
				fmt.Print("\r\n\t\t---\t\t\r\n\r\n")
			}
		}
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("===========")
}
