FROM alpine

ADD kubestat /
CMD ["/kubestat"]
