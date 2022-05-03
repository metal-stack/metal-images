#!/usr/bin/env bash

export MACHINE_TYPE=$1
export ROUTER_ID=10.1.0.1
export ASN=4200003073
export PUB_KEY="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDVh6ZujzfTzYa7Az/T1LCW3OxlpbfyjmAjfzAnBSNhq09bfj2OZZqvjiFJxXAhFmSOkzyliZHW0HED/+qFDBNhqFSwlbn8pezJKzjOe+l4pJRvRO02hguVOkzKdUM4kTWHCJOirpsRjDYVtAZ4ZVdU7stYE0C1j1JfMsSYXiHNvUmf/a+QjpCT6gdLoHvKZnDg80uBgK0GxjUOu9YaXsdypzitG8SS35gAOd1fiRV5jQk32mgue/5QQB7hgwDPRnUc5aqtxY3x4mc34VPqnW2dYHeAGftfvCHCkMYgtcBU9eGKSrJhw02xdI2MlqS5OjFErQOC0uTUXCT0dhhcfUz9GkCY+iJd1kTcHe6zVmhIBNE5WQ5oa7kBp5YWLJhQ2D6W3+1CnSKOm+1OH9QbEAMnD/IB1ljfexlViHyRijL9WRXrBi3L9Eb/Aiz7k11VUevaO8gay5UxYN4eYrebXhNpX9oW2n0NY74sMDx6QRu+lcA5VoS6z7Yh6jsqrLzM1fR8W1D8EE/MSSgMB1l3tSiJd80/hLGRUmiY3uXC8z93wZHAiR7d6L2CQSsh7KN4UgIFgzev/shvcnSTI22h68cEswVCySk8JrqP0fXTnryWS9jIzul3HSqbb3CugPnALwwJtI36tJY4Yuwu4J+FPxgxAP4AlES9GqrLJeZn6xYN5Q== cluster@footloose.mail"

if hash goss 2>/dev/null; then
    echo "goss is already installed"
else
    echo "installing goss"
    curl -kL https://github.com/aelsabbahy/goss/releases/latest/download/goss-linux-amd64 -o /usr/local/bin/goss
    chmod +rx /usr/local/bin/goss
fi

goss validate -f documentation --color --retry-timeout 60s --sleep 1s