SELECT 'CREATE DATABASE akari_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'akari_test')\gexec
