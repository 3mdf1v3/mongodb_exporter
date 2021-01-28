## Prometheus MongoDb exporter
Service install:
```
su - prometheus
mkdir prometheus_custom_exporter

cd /etc/systemd/system/
vim prometheus_custom_exporter.service

	[Unit]
	Description=RSA MongoDB Esm Exporter
	Wants=network-online.target
	After=network-online.target

	[Service]
	User=prometheus
	ExecStart=/home/prometheus/prometheus_custom_exporter/prometheus_custom_exporter -mongoDbUsername ${mongoDbUsername} -mongoDbPassword ${mongoDbPassword} -mongoDbAuthSource ${mongoDbAuthSource} -mongoDbHostURI ${mongoDbHostURI}

	[Install]
	WantedBy=default.target

systemctl daemon-reload
systemctl start prometheus_custom_exporter
systemctl enable prometheus_custom_exporter
