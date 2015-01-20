FROM busybox
MAINTAINER Alexis Montagne <alexis.montagne@gmail.com>

EXPOSE 8080

COPY fleet-ship /fleet-ship

CMD /fleet-ship
