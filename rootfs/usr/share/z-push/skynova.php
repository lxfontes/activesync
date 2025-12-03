<?php
// config.php
define('TIMEZONE', 'America/Sao_Paulo');
define('USE_FULLEMAIL_FOR_LOGIN', true);
define('IPC_PROVIDER', 'IpcSharedMemoryProvider');
define('LOGLEVEL', LOGLEVEL_DEBUG);
define('LOGAUTHFAIL', true);
define('RETRY_AFTER_DELAY', 30);
define('BACKEND_PROVIDER', 'BackendIMAP');

define('IMAP_FOLDER_CONFIGURED', true);
?>