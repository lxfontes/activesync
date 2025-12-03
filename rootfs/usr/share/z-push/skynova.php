<?php
// config.php
define('TIMEZONE', 'America/Sao_Paulo');
define('USE_FULLEMAIL_FOR_LOGIN', true);
define('IPC_PROVIDER', 'IpcSharedMemoryProvider');
define('LOGLEVEL', LOGLEVEL_DEBUG);
define('LOGAUTHFAIL', true);
define('RETRY_AFTER_DELAY', 30);
define('BACKEND_PROVIDER', 'BackendIMAP');

// autodiscover/config.php
define('AUTODISCOVER_LOGIN_TYPE', AUTODISCOVER_LOGIN_EMAIL);

// backend/ipcmemcached/config.php
define('MEMCACHED_SERVERS','localhost:11211');

// backend/imap/config.php
define('IMAP_FOLDER_CONFIGURED', true);
define('IMAP_SERVER', 'imap-ha.skymail.net.br');
define('IMAP_PORT', 993);
define('IMAP_OPTIONS', '/ssl/norsh');
define('IMAP_FOLDER_INBOX', 'INBOX');
define('IMAP_FOLDER_SENT', 'Itens Enviados');
define('IMAP_FOLDER_DRAFT', 'Rascunhos');
define('IMAP_FOLDER_TRASH', 'Itens Excluídos');
define('IMAP_FOLDER_SPAM', 'Spam');
define('IMAP_FOLDER_ARCHIVE', 'Archive');

define('IMAP_SMTP_METHOD', 'smtp');
global $imap_smtp_params;
// SMTP Parameters
//      mail : no params
$imap_smtp_params = array(
  "host" => "ssl://smtp-ha.skymail.net.br",
  "port" => 993,
  "auth" => true,
  "username" => "imap_username",
  "password" => "imap_password"
);
?>