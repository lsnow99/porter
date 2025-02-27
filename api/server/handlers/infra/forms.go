package infra

const testForm = `name: Test
hasSource: false
includeHiddenFields: true
isClusterScoped: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: String to echo
    - type: string-input
      variable: echo
      settings:
        default: hello
`

const rdsForm = `name: RDS
hasSource: false
includeHiddenFields: true
isClusterScoped: true
tabs:
- name: main
  label: Main
  sections:
  - name: heading
    contents: 
    - type: heading
      label: Database Settings
  - name: user
    contents:
    - type: string-input
      label: Database Master User
      required: true
      placeholder: "admin"
      variable: db_user
  - name: password
    contents:
    - type: string-input
      required: true
      label: Database Master Password
      variable: db_passwd
  - name: name
    contents:
    - type: string-input
      label: Database Name
      required: true
      placeholder: "rds-staging"
      variable: db_name
  - name: machine-type
    contents:
    - type: select
      label: ⚙️ Database Machine Type
      variable: machine_type
      settings:
        default: db.t3.medium
        options:
        - label: db.t2.medium
          value: db.t2.medium
        - label: db.t2.xlarge
          value: db.t2.xlarge
        - label: db.t2.2xlarge
          value: db.t2.2xlarge
        - label: db.t3.medium
          value: db.t3.medium
        - label: db.t3.xlarge
          value: db.t3.xlarge
        - label: db.t3.2xlarge
          value: db.t3.2xlarge
  - name: family-versions
    contents:
    - type: select
      label:  Database Family Version
      variable: db_family
      settings:
        default: postgres13
        options:
        - label: "Postgres 9"
          value: postgres9
        - label: "Postgres 10"
          value: postgres10
        - label: "Postgres 11"
          value: postgres11
        - label: "Postgres 12"
          value: postgres12
        - label: "Postgres 13"
          value: postgres13
  - name: pg-9-versions
    show_if: 
      is: "postgres9"
      variable: db_family
    contents:
    - type: select
      label:  Database Version
      variable: db_engine_version
      settings:
        default: "9.6.23"
        options:
        - label: "v9.6.1"
          value: "9.6.1"
        - label: "v9.6.2"
          value: "9.6.2"
        - label: "v9.6.3"
          value: "9.6.3"
        - label: "v9.6.4"
          value: "9.6.4"
        - label: "v9.6.5"
          value: "9.6.5"
        - label: "v9.6.6"
          value: "9.6.6"
        - label: "v9.6.7"
          value: "9.6.7"
        - label: "v9.6.8"
          value: "9.6.8"
        - label: "v9.6.10"
          value: "9.6.10"
        - label: "v9.6.11"
          value: "9.6.11"
        - label: "v9.6.12"
          value: "9.6.12"
        - label: "v9.6.13"
          value: "9.6.13"
        - label: "v9.6.14"
          value: "9.6.14"
        - label: "v9.6.15"
          value: "9.6.15"
        - label: "v9.6.16"
          value: "9.6.16"
        - label: "v9.6.17"
          value: "9.6.17"
        - label: "v9.6.18"
          value: "9.6.18"
        - label: "v9.6.19"
          value: "9.6.19"
        - label: "v9.6.20"
          value: "9.6.20"
        - label: "v9.6.21"
          value: "9.6.21"
        - label: "v9.6.22"
          value: "9.6.22"
        - label: "v9.6.23"
          value: "9.6.23"
  - name: pg-10-versions
    show_if: 
      is: "postgres10"
      variable: db_family
    contents:
    - type: select
      label:  Database Version
      variable: db_engine_version
      settings:
        default: "10.18"
        options:
        - label: "v10.1"
          value: "10.1"
        - label: "v10.2"
          value: "10.2"
        - label: "v10.3"
          value: "10.3"
        - label: "v10.4"
          value: "10.4"
        - label: "v10.5"
          value: "10.5"
        - label: "v10.6"
          value: "10.6"
        - label: "v10.7"
          value: "10.7"
        - label: "v10.8"
          value: "10.8"
        - label: "v10.9"
          value: "10.9"
        - label: "v10.10"
          value: "10.10"
        - label: "v10.11"
          value: "10.11"
        - label: "v10.12"
          value: "10.12"
        - label: "v10.13"
          value: "10.13"
        - label: "v10.14"
          value: "10.14"
        - label: "v10.15"
          value: "10.15"
        - label: "v10.16"
          value: "10.16"
        - label: "v10.17"
          value: "10.17"
        - label: "v10.18"
          value: "10.18"
  - name: pg-11-versions
    show_if: 
      is: "postgres11"
      variable: db_family
    contents:
    - type: select
      label:  Database Version
      variable: db_engine_version
      settings:
        default: "11.13"
        options:
        - label: "v11.1"
          value: "11.1"
        - label: "v11.2"
          value: "11.2"
        - label: "v11.3"
          value: "11.3"
        - label: "v11.4"
          value: "11.4"
        - label: "v11.5"
          value: "11.5"
        - label: "v11.6"
          value: "11.6"
        - label: "v11.7"
          value: "11.7"
        - label: "v11.8"
          value: "11.8"
        - label: "v11.9"
          value: "11.9"
        - label: "v11.10"
          value: "11.10"
        - label: "v11.11"
          value: "11.11"
        - label: "v11.12"
          value: "11.12"
        - label: "v11.13"
          value: "11.13"
  - name: pg-12-versions
    show_if: 
      is: "postgres12"
      variable: db_family
    contents:
    - type: select
      label:  Database Version
      variable: db_engine_version
      settings:
        default: "12.8"
        options:
        - label: "v12.2"
          value: "12.2"
        - label: "v12.3"
          value: "12.3"
        - label: "v12.4"
          value: "12.4"
        - label: "v12.5"
          value: "12.5"
        - label: "v12.6"
          value: "12.6"
        - label: "v12.7"
          value: "12.7"
        - label: "v12.8"
          value: "12.8"
  - name: pg-13-versions
    show_if: 
      is: "postgres13"
      variable: db_family
    contents:
    - type: select
      label:  Database Version
      variable: db_engine_version
      settings:
        default: "13.4"
        options:
        - label: "v13.1"
          value: "13.1"
        - label: "v13.2"
          value: "13.2"
        - label: "v13.3"
          value: "13.3"
        - label: "v13.4"
          value: "13.4"
  - name: additional-settings
    contents:
    - type: heading
      label: Additional Settings
    - type: checkbox
      variable: db_deletion_protection
      label: Enable deletion protection for the database.
      settings:
        default: false
- name: storage
  label: Storage
  sections:
  - name: storage
    contents:
    - type: heading
      label: Storage Settings
    - type: number-input
      label: Gigabytes
      variable: db_allocated_storage
      placeholder: "ex: 10"
      settings:
        default: 10
    - type: number-input
      label: Gigabytes
      variable: db_max_allocated_storage
      placeholder: "ex: 20"
      settings:
        default: 20
    - type: checkbox
      variable: db_storage_encrypted
      label: Enable storage encryption for the database. 
      settings:
        default: false`

const ecrForm = `name: ECR
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: ECR Configuration
    - type: string-input
      label: ECR Name
      required: true
      placeholder: my-awesome-registry
      variable: ecr_name
`

const eksForm = `name: EKS
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: EKS Configuration
    - type: select
      label: ⚙️ AWS Machine Type
      variable: machine_type
      settings:
        default: t2.medium
        options:
        - label: t2.medium
          value: t2.medium
    - type: string-input
      label: 👤 Issuer Email
      required: true
      placeholder: example@example.com
      variable: issuer_email
    - type: string-input
      label: EKS Cluster Name
      required: true
      placeholder: my-cluster
      variable: cluster_name
`

const gcrForm = `name: GCR
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: GCR Configuration
    - type: select
      label: 📍 GCP Region
      variable: gcp_region
      settings:
        default: us-central1
        options:
        - label: asia-east1
          value: asia-east1
        - label: asia-east2
          value: asia-east2
        - label: asia-northeast1
          value: asia-northeast1
        - label: asia-northeast2
          value: asia-northeast2
        - label: asia-northeast3
          value: asia-northeast3
        - label: asia-south1
          value: asia-south1
        - label: asia-southeast1
          value: asia-southeast1
        - label: asia-southeast2
          value: asia-southeast2
        - label: australia-southeast1
          value: australia-southeast1
        - label: europe-north1
          value: europe-north1
        - label: europe-west1
          value: europe-west1
        - label: europe-west2
          value: europe-west2
        - label: europe-west3
          value: europe-west3
        - label: europe-west4
          value: europe-west4
        - label: europe-west6
          value: europe-west6
        - label: northamerica-northeast1
          value: northamerica-northeast1
        - label: southamerica-east1
          value: southamerica-east1
        - label: us-central1
          value: us-central1
        - label: us-east1
          value: us-east1
        - label: us-east4
          value: us-east4
        - label: us-east1
          value: us-east1
        - label: us-east1
          value: us-east1
        - label: us-west1
          value: us-west1
        - label: us-east1
          value: us-west2
        - label: us-west3
          value: us-west3
        - label: us-west4
          value: us-west4
`

const gkeForm = `name: GKE
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: GKE Configuration
    - type: select
      label: 📍 GCP Region
      variable: gcp_region
      settings:
        default: us-central1
        options:
        - label: asia-east1
          value: asia-east1
        - label: asia-east2
          value: asia-east2
        - label: asia-northeast1
          value: asia-northeast1
        - label: asia-northeast2
          value: asia-northeast2
        - label: asia-northeast3
          value: asia-northeast3
        - label: asia-south1
          value: asia-south1
        - label: asia-southeast1
          value: asia-southeast1
        - label: asia-southeast2
          value: asia-southeast2
        - label: australia-southeast1
          value: australia-southeast1
        - label: europe-north1
          value: europe-north1
        - label: europe-west1
          value: europe-west1
        - label: europe-west2
          value: europe-west2
        - label: europe-west3
          value: europe-west3
        - label: europe-west4
          value: europe-west4
        - label: europe-west6
          value: europe-west6
        - label: northamerica-northeast1
          value: northamerica-northeast1
        - label: southamerica-east1
          value: southamerica-east1
        - label: us-central1
          value: us-central1
        - label: us-east1
          value: us-east1
        - label: us-east4
          value: us-east4
        - label: us-east1
          value: us-east1
        - label: us-east1
          value: us-east1
        - label: us-west1
          value: us-west1
        - label: us-east1
          value: us-west2
        - label: us-west3
          value: us-west3
        - label: us-west4
          value: us-west4
    - type: string-input
      label: 👤 Issuer Email
      required: true
      placeholder: example@example.com
      variable: issuer_email
    - type: string-input
      label: GKE Cluster Name
      required: true
      placeholder: my-cluster
      variable: cluster_name
`

const docrForm = `name: DOCR
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: DOCR Configuration
    - type: select
      label: DO Subscription Tier
      variable: docr_subscription_tier
      settings:
        default: basic
        options:
        - label: Basic
          value: basic
        - label: Professional
          value: professional
    - type: string-input
      label: DOCR Name
      required: true
      placeholder: my-awesome-registry
      variable: docr_name
`

const doksForm = `name: DOKS
hasSource: false
includeHiddenFields: true
tabs:
- name: main
  label: Configuration
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: DOKS Configuration
    - type: select
      label: 📍 DO Region
      variable: do_region
      settings:
        default: nyc1
        options:
        - label: Amsterdam 3
          value: ams3
        - label: Bangalore 1
          value: blr1
        - label: Frankfurt 1
          value: fra1
        - label: London 1
          value: lon1
        - label: New York 1
          value: nyc1
        - label: New York 3
          value: nyc3
        - label: San Francisco 2
          value: sfo2
        - label: San Francisco 3
          value: sfo3
        - label: Singapore 1
          value: sgp1
        - label: Toronto 1
          value: tor1
    - type: string-input
      label: 👤 Issuer Email
      required: true
      placeholder: example@example.com
      variable: issuer_email
    - type: string-input
      label: DOKS Cluster Name
      required: true
      placeholder: my-cluster
      variable: cluster_name
`
