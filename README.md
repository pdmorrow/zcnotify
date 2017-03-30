zcnotify.
---------

Sends notifications via email when services join/update/leave the zeroconf multicast group.  Notication options are specified in a TOML configuration file, an example config file looks like this:

	ScanPeriodSeconds = 5               # Check for changes every 5 seconds.
	NotifyTypes = ["email"]             # Send notifications via email only.

	[zeroconf]
	Service = "_workstation._tcp"
	Domain = "local"

	[interfaces]
	Exclude = ["lo", "docker0"]
	Ip = ["ipv4", "ipv6"]               # Join both v4 & v6 multicast groups.

	[email]
    	[email.pdmorrow]                # Send emails to this address.
    	From = "pdmorrow@gmail.com"
    	To = "pdmorrow@gmail.com"
    	Ssl = true
    	Server = "smtp.gmail.com:587"
    	Password = "???"
