package cli

import (
	"bufio"
	"fmt"
	"geoservice/geo"
	"geoservice/messaging"
	"github.com/rs/xid"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
)

const geoRegexp = `^geo-.*`

type Commander struct {
	broker messaging.Broker
	name   string
}

func NewCommander(b messaging.Broker) *Commander {
	return &Commander{
		broker: b,
		name:   fmt.Sprintf("cli-%s", xid.New().String()),
	}
}

func (c *Commander) Start() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Commands:\n1. Find closest geo service\n2. Update geo services' positions\n> ")
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			continue
		}
		str = strings.ReplaceAll(str, "\n", "")
		cmd, err := strconv.Atoi(str)
		if err != nil {
			log.Println(err)
			continue
		}
		switch cmd {
		case 1:
			fmt.Print("Enter X and Y coordinates separated by comma: ")
			str, err = reader.ReadString('\n')
			if err != nil {
				log.Println(err)
				continue
			}
			values := strings.Split(str, ",")
			if len(values) < 2 {
				fmt.Println("2 values expected")
				continue
			}
			x, err1 := strconv.Atoi(strip(values[0]))
			y, err2 := strconv.Atoi(strip(values[1]))
			if err1 != nil || err2 != nil {
				fmt.Printf("please check your numbers (%s, %s)", err1, err2)
				continue
			}
			srvName := c.closest(x, y)
			fmt.Printf("=> %s\n\n", srvName)
		case 2:
			msg := messaging.Message{
				Body: geo.Request{Method: geo.MethodChangePosition},
			}
			c.broker.Broadcast(geoRegexp, msg)
		case 3:
			c.broker.GetServices()
		default:
			continue
		}
	}
}

func (c *Commander) closest(x, y int) string {
	services := c.broker.GetServices()
	// produce names
	namesChan := make(chan string)
	go func() {
		for _, s := range services {
			namesChan <- s
		}
	}()
	// distribute requests
	msg := messaging.Message{
		Body: geo.Request{
			Method: geo.MethodGetDistance,
			X:      float64(x),
			Y:      float64(y),
		},
		Sender: c.name,
	}
	wg := sync.WaitGroup{}
	wg.Add(len(services))
	resCh := make(chan geo.ServiceDist, len(services))
	for i := 0; i < 3; i++ {
		go func() {
			for name := range namesChan {
				func() {
					defer wg.Done()
					replyCh := c.broker.Send(name, msg)
					reply := <-replyCh
					if reply.Error != nil {
						log.Println(reply.Error)
						return
					}
					dist, ok := reply.Body.(geo.ServiceDist)
					if !ok {
						log.Printf("failed convert %v to geo.ServiceDist\n", reply.Body)
						return
					}
					resCh <- dist
				}()
			}
		}()
	}

	// collect results
	done := make(chan struct{})
	closestService := geo.ServiceDist{Dist: math.MaxFloat64}
	go func() {
		for serviceDist := range resCh {
			if closestService.Dist < serviceDist.Dist {
				closestService = serviceDist
			}
		}
		done <- struct{}{}
	}()

	wg.Wait()
	close(namesChan)
	close(resCh)
	<-done

	return closestService.Service
}

func strip(str string) string {
	return strings.ReplaceAll(str, "\n", "")
}
