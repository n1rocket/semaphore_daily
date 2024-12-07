openssl req -x509 -newkey rsa:2048 -days 3650 -nodes   -keyout ca.key -out ca.crt -subj "/CN=Internal-Root-CA"
kubectl create secret tls internal-ca-secret   --cert=ca.crt   --key=ca.key   -n registry
cat << EOF > openssl.conf
[ req ]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[ req_distinguished_name ]

[ v3_req ]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
IP.1 = 192.168.1.222
EOF
