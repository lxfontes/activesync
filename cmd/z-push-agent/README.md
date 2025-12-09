# z-push-agent

Este servico roda em todos servidores Z-Push. Comandos sao executados apenas nos servidores responsaveis por cada usuario.

Monitora mensagens Redis nos topicos `activesync.command.*` no formato `usuario@dominio device-id`. Por exemplo:

```
PUBLISH activesync.command.resync "lucas@ghz.com.br n6fok02kb14bda532uoiodepq4"
```

## Comandos

### `resync`

Manda o device re-sincronizar todos emails/contatos/calendarios.

```
PUBLISH activesync.command.resync "lucas@ghz.com.br n6fok02kb14bda532uoiodepq4"
```

### `clearloop`

Raramente necessario. O device pode rejeitar uma mensagem/contato/calendario por falha de encoding (ex: UTF8).

```
PUBLISH activesync.command.clearloop "lucas@ghz.com.br n6fok02kb14bda532uoiodepq4"
```

Essa situacao se apresenta nos Logs como:

```
    Information:    Subject: 'Good day!' - From: '<lxf@31337.dev>'
    Reason:         Message was causing loop (2)
    Item/Parent id: 31222/i/33756155
```

### `fixstates`

Raramente necessario. Verifica e reconstroi estruturas internas apos upgrades de versao do Z-Push.

```
PUBLISH activesync.command.clearloop "lucas@ghz.com.br n6fok02kb14bda532uoiodepq4"
```

### `remove`

Remove todos dados do usuario+device. Diferente de `resync`, esta chamada tb apaga o device token usado para operacoes de remote wipe.

```
PUBLISH activesync.command.remove "lucas@ghz.com.br n6fok02kb14bda532uoiodepq4"
```