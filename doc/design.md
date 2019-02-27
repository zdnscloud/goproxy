# Table of Contents #

- [Requirement](#requirement)
- [Components](#componenets)
- [Data Flow](#data-flow)

-----------------------------

# Requirement #
To consume the unexposed service in k8s cluster, tunnel needs
to be built between the client network and k8s network. Normally
client will use explorer to interact with the backend service. 

# Component #
- Client: the consumer for the service
- Server: the gateway between client and backend service
- Agent: proxy lived in the same network with the service

# Data Flow #
- At first, agent will register itself with server, and a ws connection 
is maintained by both side.

- Client connect to server, tell the server which agent and service they
want to consume.

- Server find the the specified agent, and send connect request to the agent 
to tell which service to connect. Since several client could use 
the same agent, so for each client connection, server and agent create 
a related fake connection for the client. 

- For both Server and Agent, the ws connection and all the related fake 
connection is called a session.

- Agent associate the fake connection with the connection to the real 
service, then use go routine to pipe the fake connection with the real 
connection.

- At this stage, the virtual connection is built between Client and the
real service: 
  - Client --> http/ws --> Server --> ws --> Agent --> UDP/TCP/HTTP/ws --> Service

- Client get the fake connection from the Server. When Client send data 
to fake connection, Server will copy the data to the ws connection related 
to the agent and specify the connection id for current Client. 

- Agent get the message from ws connection, find the fake connection, 
copy the data to it. the pipe routine will copy the data to the real 
connection, the response from the real connection will be copied to the 
fake connection and then relay to the ws connection with the connection id 

- Finally server will copy the data to the fake connection which client 
could read from.
