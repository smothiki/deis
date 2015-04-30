# security keys and auth tokens
SECRET_KEY = '{{ getv "/deis/controller/secretKey" }}'
BUILDER_KEY = '{{ getv "/deis/controller/builderKey" }}'

# scheduler settings
SCHEDULER_MODULE = 'scheduler.{{ if exists "/deis/controller/schedulerModule" }}{{ getv "/deis/controller/schedulerModule" }}{{ else }}fleet{{ end }}'
SCHEDULER_TARGET = '{{ if exists "/deis/controller/schedulerTarget" }}{{ getv "/deis/controller/schedulerTarget" }}{{ else }}/var/run/fleet.sock{{ end }}'
try:
    SCHEDULER_OPTIONS = dict('{{ if exists "/deis/controller/schedulerOptions" }}{{ getv "/deis/controller/schedulerOptions" }}{{ else }}{}{{ end }}')
except:
    SCHEDULER_OPTIONS = {}

# scheduler swarm manager host
SWARM_HOST = '{{ if exists "/deis/scheduler/swarm/host" }}{{ getv "/deis/scheduler/swarm/host" }}{{ else }}127.0.0.1{{ end }}'
#SWARM_HOST = '{{ or (.deis_scheduler_swarm_host) "127.0.0.1" }}'


# base64-encoded SSH private key to facilitate current version of "deis run"
SSH_PRIVATE_KEY = """{{ if exists "/deis/platform/sshPrivateKey" }}{{ getv "/deis/platform/sshPrivateKey" }}{{ else }}""{{end}}"""

# platform domain must be provided
DEIS_DOMAIN = '{{ getv "/deis/platform/domain" }}'

# use the private registry module
REGISTRY_MODULE = 'registry.private'
REGISTRY_URL = '{{ getv "/deis/registry/protocol" }}://{{ getv "/deis/registry/host" }}:{{ getv "/deis/registry/port" }}'  # noqa
REGISTRY_HOST = '{{ getv "/deis/registry/host" }}'
REGISTRY_PORT = '{{ getv "/deis/registry/port" }}'

# default to sqlite3, but allow postgresql config through envvars
DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.{{ getv "/deis/database/engine" }}',
        'NAME': '{{ getv "/deis/database/name" }}',
        'USER': '{{ getv "/deis/database/user" }}',
        'PASSWORD': '{{ getv "/deis/database/password" }}',
        'HOST': '{{ getv "/deis/database/host" }}',
        'PORT': '{{ getv "/deis/database/port" }}',
    }
}

# move log directory out of /app/deis
DEIS_LOG_DIR = '/data/logs'

{{ if exists "/deis/controller/registrationMode" }}
REGISTRATION_MODE = '{{ getv "/deis/controller/registrationMode" }}'
{{ end }}

{{ if exists "/deis/controller/webEnabled" }}
WEB_ENABLED = bool({{ getv "/deis/controller/webEnabled" }})
{{ end }}
UNIT_HOSTNAME = '{{ if exists "/deis/controller/unitHostname" }}{{ getv "/deis/controller/unitHostname" }}{{ else }}default{{ end }}'

# AUTH
# LDAP
{{ if exists "/deis/controller/auth/ldap/endpoint" }}
LDAP_ENDPOINT = '{{ if exists "/deis/controller/auth/ldap/endpoint" }}{{ getv "/deis/controller/auth/ldap/endpoint"}}{{ else }} {{ end }}'
BIND_DN = '{{ if exists "/deis/controller/auth/ldap/bind/dn" }}{{ getv "/deis/controller/auth/ldap/bind/dn"}}{{ else }} {{ end }}'
BIND_PASSWORD = '{{ if exists "/deis/controller/auth/ldap/bind/password" }}{{ getv "/deis/controller/auth/ldap/bind/password"}}{{ else }} {{ end }}'
USER_BASEDN = '{{ if exists "/deis/controller/auth/ldap/user/basedn" }}{{ getv "/deis/controller/auth/ldap/user/basedn"}}{{ else }} {{ end }}'
USER_FILTER = '{{ if exists "/deis/controller/auth/ldap/user/filter" }}{{ getv "/deis/controller/auth/ldap/user/filter"}}{{ else }} {{ end }}'
GROUP_BASEDN = '{{ if exists "/deis/controller/auth/ldap/group/basedn" }}{{ getv "/deis/controller/auth/ldap/group/basedn"}}{{ else }} {{ end }}'
GROUP_FILTER = '{{ if exists "/deis/controller/auth/ldap/group/filter" }}{{ getv "/deis/controller/auth/ldap/group/filter"}}{{ else }} {{ end }}'
GROUP_TYPE = '{{ if exists "/deis/controller/auth/ldap/group/type" }}{{ getv "/deis/controller/auth/ldap/group/type"}}{{ else }} {{ end }}'
{{ end }}
