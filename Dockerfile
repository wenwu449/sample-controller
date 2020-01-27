FROM ubuntu:18.04
COPY ./crdapp-controller /crdapp-controller
CMD bash -c "/crdapp-controller"