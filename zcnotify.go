// zcnotify generates simple reports whenever zeroconf based services
// appear/disappear/change on the network.  Notification parameters are
// specified via a TOML file
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/grandcat/zeroconf"
)

// interfaceNames Returns a list of interface names given a list of
// net.Interface objects.
func interfaceNames(intfs []net.Interface) []string {
	var intfNames []string

	for _, intf := range intfs {
		intfNames = append(intfNames, intf.Name)
	}

	return intfNames
}

// compareSEKey Compares the key parts of a zeroconf.ServiceEntry.
func compareSEKey(a *zeroconf.ServiceEntry, b *zeroconf.ServiceEntry) bool {
	return a.ServiceInstanceName() == b.ServiceInstanceName()
}

// compareSEEntry Compares the payload of a zeroconf.ServiceEntry.
func compareSEEntry(a *zeroconf.ServiceEntry, b *zeroconf.ServiceEntry) bool {
	if a.HostName != b.HostName {
		return false
	}

	if a.Port != b.Port {
		return false
	}

	if a.TTL != b.TTL {
		return false
	}

	if len(a.Text) != len(b.Text) {
		return false
	} else {
		for _, aEntry := range a.Text {
			for _, bEntry := range b.Text {
				if aEntry != bEntry {
					return false
				}
			}
		}
	}

	if len(a.AddrIPv4) != len(b.AddrIPv4) {
		return false
	} else {
		for _, aAddr := range a.AddrIPv4 {
			for _, bAddr := range b.AddrIPv4 {
				if !aAddr.Equal(bAddr) {
					return false
				}
			}
		}
	}

	if len(a.AddrIPv6) != len(b.AddrIPv6) {
		return false
	} else {
		for _, aAddr := range a.AddrIPv6 {
			for _, bAddr := range b.AddrIPv6 {
				if !aAddr.Equal(bAddr) {
					return false
				}
			}
		}
	}

	return true
}

// watchZCGroups periodically browses the zeroconf multicast group(s) and notifies
// group change events via the updates channel.
func watchZCGroups(done chan error,
	exit chan bool,
	updates chan ServiceEntryChange,
	service string,
	domain string,
	periodSecs uint,
	ipver zeroconf.IPType,
	intfs []net.Interface) {
	var previousEntries []zeroconf.ServiceEntry

	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond):
			// Wake up and browse the multicast group(s).
			break
		case <-exit:
			// Received the exit signal from the exit channel.
			done <- nil
			return
		}

		resolver, err := zeroconf.NewResolver(zeroconf.SelectIPTraffic(ipver),
			zeroconf.SelectIfaces(intfs))

		if err != nil {
			log.Fatalln("Failed to initialize resolver:", err.Error())
			done <- err
			return
		}

		entries := make(chan *zeroconf.ServiceEntry)
		go func(results <-chan *zeroconf.ServiceEntry,
			prev *[]zeroconf.ServiceEntry) {
			// Look at each result, if we've not seen this service before
			// then signal an ADD via the update channel.
			var entries []zeroconf.ServiceEntry
			for entry := range results {
				new_entry := true
				for _, old_entry := range *prev {
					if compareSEKey(&old_entry, entry) {
						new_entry = false
						if !compareSEEntry(&old_entry, entry) {
							updates <- ServiceEntryChange{MODIFY,
								time.Now().UTC(), *entry}
						}

						break
					}
				}

				if new_entry {
					*prev = append(*prev, *entry)
					updates <- ServiceEntryChange{ADD, time.Now().UTC(), *entry}
				}

				entries = append(entries, *entry)
			}

			// Check if any of the old services were not in this update, if
			// a service has gone then signal a REMOVE via the update channel.
			for index := len(*prev) - 1; index >= 0; index-- {
				found := false
				for _, entry := range entries {
					if compareSEKey(&entry, &((*prev)[index])) {
						found = true
						break
					}
				}

				if !found {
					updates <- ServiceEntryChange{REMOVE,
						time.Now().UTC(),
						(*prev)[index]}
					*prev = append((*prev)[:index], (*prev)[index+1:]...)
				}
			}
		}(entries, &previousEntries)

		// Browse the group(s), updates are delivered via the entries channel
		// and thus the anonymous goroutine above will be called to process
		// found entries.
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Second*time.Duration(periodSecs))
		err = resolver.Browse(ctx, service, domain, entries)
		<-ctx.Done()
		cancel()
		if err != nil {
			log.Fatalln("Failed to browse:", err.Error())
			done <- err
			return
		}
	}
}

func main() {
	var (
		ipver      zeroconf.IPType
		intfs      []net.Interface
		err        error
		zcnConfig  config
		configFile = flag.String("config",
			"zcnotify.toml",
			"Configuration TOML file")
	)

	flag.Parse()
	// Decode and parse the supplied config, if no config exists use sensible
	// defaults.
	if _, err := toml.DecodeFile(*configFile, &zcnConfig); err != nil {
		log.Fatalln("failed to decode config file:", err.Error())
	} else {
		if len(zcnConfig.Interfaces.Ip) == 0 {
			// Default to v4 and v6 if not specified.
			ipver = zeroconf.IPv4AndIPv6
		} else {
			// Which ip versions can we use on the local discovery interfaces?
			for _, ipv := range zcnConfig.Interfaces.Ip {
				switch ipv {
				case "ipv4":
					ipver |= zeroconf.IPv4
					break
				case "ipv6":
					ipver |= zeroconf.IPv6
					break
				default:
					log.Fatalf("unknown IP version %s in interface config", ipv)
				}
			}
		}
	}

	if len(zcnConfig.Interfaces.Use) == 0 {
		// No interfaces specified, use all.
		intfs, err = net.Interfaces()
		if err != nil {
			log.Fatalln("cannot retrieve system interfaces:", err.Error())
		}
		log.Println("no interfaces specified, assuming all:",
			interfaceNames(intfs))
	} else {
		for _, intfName := range zcnConfig.Interfaces.Use {
			intf, err := net.InterfaceByName(intfName)
			if err != nil {
				log.Fatalf("no such interface %q", intfName)
			} else {
				intfs = append(intfs, *intf)
			}
		}

		log.Println("using specific interfaces", interfaceNames(intfs))
	}

	if len(zcnConfig.Interfaces.Exclude) > 0 {
		log.Printf("excluding interfaces %s", zcnConfig.Interfaces.Exclude)
		for _, excludeIntfName := range zcnConfig.Interfaces.Exclude {
			_, err := net.InterfaceByName(excludeIntfName)
			if err != nil {
				log.Fatalf("no such interface %q", excludeIntfName)
			} else {
				for index := len(intfs) - 1; index >= 0; index-- {
					if excludeIntfName == intfs[index].Name {
						intfs = append(intfs[:index], intfs[index+1:]...)
					}
				}
			}
		}
	}

	log.Println("final interface list", interfaceNames(intfs))

	if zcnConfig.Zeroconf.Service == "" {
		zcnConfig.Zeroconf.Service = DEFAULT_SERVICE
	} else if zcnConfig.Zeroconf.Service != DEFAULT_SERVICE {
		log.Fatalln("unknown zeroconf service:", zcnConfig.Zeroconf.Service)
	}

	if zcnConfig.Zeroconf.Domain == "" {
		zcnConfig.Zeroconf.Domain = DEFAULT_DOMAIN
	} else if zcnConfig.Zeroconf.Domain != DEFAULT_DOMAIN {
		log.Fatalln("unknown zeroconf domain:", zcnConfig.Zeroconf.Domain)
	}

	if zcnConfig.ScanPeriodSeconds == 0 {
		zcnConfig.ScanPeriodSeconds = DEFAULT_SCAN_PERIOD
	}

	log.Printf("will browse every %d seconds", zcnConfig.ScanPeriodSeconds)

	if len(zcnConfig.NotifyTypes) == 0 {
		log.Fatalln("no notification types found in config file")
	}

	for _, notifyType := range zcnConfig.NotifyTypes {
		notifyTypeLower := strings.ToLower(notifyType)
		switch notifyTypeLower {
		case "email":
			if err := ValidEmailConfig(zcnConfig.Email); err != nil {
				log.Fatalln("invalid email configuration settings:",
					err.Error())
			}
			break
		default:
			log.Fatalf("unknown notification type %q", notifyTypeLower)
		}
	}

	// Done parsing the config file.
	done := make(chan error, 1)
	exit := make(chan bool, 1)
	updates := make(chan ServiceEntryChange, 1)

	// Process newly discovered or removed services.
	go func(updates chan ServiceEntryChange, zConfig *config) {
		for {
			change := <-updates
			for _, notifyType := range zConfig.NotifyTypes {
				switch notifyType {
				case "email":
					go SendEmail(zcnConfig.Email, &change)
					break
				default:
					panic(fmt.Sprintf("unknown notification type %q", notifyType))
				}
			}
		}
	}(updates, &zcnConfig)

	// Watch for changes to the multicast groups by browsing periodically.
	go watchZCGroups(done,
		exit,
		updates,
		zcnConfig.Zeroconf.Service,
		zcnConfig.Zeroconf.Domain,
		zcnConfig.ScanPeriodSeconds,
		ipver,
		intfs)

	// Handle interrupt signals, on receiving one deliver a notification
	// to the watchZCGroups goroutine so it terminates.
	sigchan := make(chan os.Signal, 1)
	go func() {
		<-sigchan
		log.Println("interrupt received")
		exit <- true
	}()

	signal.Notify(sigchan, os.Interrupt)

	// Wait till the watchZCGroups goroutine exits, either via an error or
	// via an interrupt signal.
	watchZCGroupsErr := <-done
	if watchZCGroupsErr != nil {
		log.Fatalln("exited:", watchZCGroupsErr.Error())
	} else {
		log.Println("exited")
	}
}
