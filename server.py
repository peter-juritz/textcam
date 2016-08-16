import SocketServer
import time	
clients = {}
class ClientHandler(SocketServer.BaseRequestHandler):
    def handle(self):
        client_request = self.request.recv(500).strip()
        p = client_request.split('\n')
        client_name = p[0]
        target_name = p[1]
        print "{} wrote:".format(self.client_address[0])
        clients[client_name] = self.request   

        while p[1] not in clients:
            print 'Waiting for client'
            time.sleep(1)
        self.request.send("Ready\n")

        while True:
            data = self.request.recv(3500).strip()
            print 'Got data packets from %s : %s' % (client_name , len(data))
            clients[target_name].send(data)
            #print 'Sent them to  : %s' % (target_name)
            
class ThreadingTCPServer(SocketServer.ThreadingMixIn, SocketServer.TCPServer):
    pass

if __name__ == "__main__":
    HOST, PORT = "localhost", 8081

    server = ThreadingTCPServer((HOST, PORT), ClientHandler)
    server.serve_forever()
