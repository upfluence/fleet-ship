FROM busybox
MAINTAINER Alexis Montagne <alexis.montagne@gmail.com>

EXPOSE 8080

COPY etcdenv /etcdenv
COPY fleet-ship /fleet-ship

RUN chmod +x /etcdenv
RUN chmod +x /fleet-ship

ENV NAMESPACE /environments/global

CMD /etcdenv -n ${NAMESPACE} -s http://172.17.42.1:4001 /fleet-ship
