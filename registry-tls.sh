kubectl create secret tls registry-tls \
  --cert=tls.crt --key=tls.key


kubectl create configmap registry-ca --from-file=ca.crt=tls.crt
