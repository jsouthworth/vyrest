# vyrest
A Go library to talk to a Vyatta router via its REST API

Based on the documentation for the REST API here:
[Vyatta REST](http://0.us.mirrors.vyos.net/vyatta/vc6.5/docs/Vyatta-Documentation_6.5R1_v01/Vyatta-RemoteAccessAPI2.0_6.5R1_v01.pdf)

Examples:
```
~$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta run-cmd show interfaces
Codes: S - State, L - Link, u - Up, D - Down, A - Admin Down
Interface       IP Address                        S/L  Speed/Duplex  Description
---------       ----------                        ---  ------------  -----------
dp0p33p1        192.168.178.139/24                u/u  a-1g/a-full
dp0p34p1        -                                 u/u  a-1g/a-full
dp0p35p1        -                                 u/u  a-1g/a-full
dp0p36p1        -                                 A/D  auto/auto

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta setup-session
54BB0EDDCA50661E

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta list-sessions
session-id		username	description
----------		--------	-----------
8BE9899F1FDFB0ED	vyatta
54BB0EDDCA50661E	vyatta

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta teardown-session 8BE9899F1FDFB0ED

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta -sid 54BB0EDDCA50661E set interfaces dataplane dp0p33p1 description "Management Interface"

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta -sid 54BB0EDDCA50661E show
 interfaces {
 	dataplane dp0p33p1 {
 		address dhcp
 		description "Management Interface"
 	}
	...

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta -sid 54BB0EDDCA50661E commit

$ vyrest -host 192.168.178.139 -user vyatta -pass vyatta run-cmd show interfaces
Codes: S - State, L - Link, u - Up, D - Down, A - Admin Down
Interface       IP Address                        S/L  Speed/Duplex  Description
---------       ----------                        ---  ------------  -----------
dp0p33p1        192.168.178.139/24                u/u  a-1g/a-full   Management
                                                                     Interface
dp0p34p1        -                                 u/u  a-1g/a-full
dp0p35p1        -                                 u/u  a-1g/a-full
dp0p36p1        -                                 A/D  auto/auto

```
