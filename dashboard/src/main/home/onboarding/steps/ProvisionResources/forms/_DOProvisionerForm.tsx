import Helper from "components/form-components/Helper";
import InputRow from "components/form-components/InputRow";
import SelectRow from "components/form-components/SelectRow";
import SaveButton from "components/SaveButton";
import { OFState } from "main/home/onboarding/state";
import { DOProvisionerConfig } from "main/home/onboarding/types";
import React, { useEffect, useState } from "react";
import api from "shared/api";
import styled from "styled-components";
import { useSnapshot } from "valtio";
import Loading from "components/Loading";
import { readableDate } from "shared/string_utils";
import { Infrastructure } from "shared/types";

const tierOptions = [
  { value: "basic", label: "Basic" },
  { value: "professional", label: "Professional" },
];

const regionOptions = [
  { value: "ams3", label: "Amsterdam 3" },
  { value: "blr1", label: "Bangalore 1" },
  { value: "fra1", label: "Frankfurt 1" },
  { value: "lon1", label: "London 1" },
  { value: "nyc1", label: "New York 1" },
  { value: "nyc3", label: "New York 3" },
  { value: "sfo2", label: "San Francisco 2" },
  { value: "sfo3", label: "San Francisco 3" },
  { value: "sgp1", label: "Singapore 1" },
  { value: "tor1", label: "Toronto 1" },
];

/**
 * This will redirect to DO, and we should pass the redirection URI to be /onboarding/provision?provider=do
 *
 * After the oauth flow comes back, the first render will go and check if it exists a integration_id for DO in the
 * current onboarding project, after getting it, the CredentialsForm will use nextFormStep to save the onboarding state.
 *
 * If it happens to be an error, it will be shown with the default error handling through the modal.
 */
export const CredentialsForm: React.FC<{
  nextFormStep: (data: Partial<DOProvisionerConfig>) => void;
  project: any;
}> = ({ nextFormStep, project }) => {
  const snap = useSnapshot(OFState);

  const [isLoading, setIsLoading] = useState(true);
  const [connectedAccount, setConnectedAccount] = useState(null);

  useEffect(() => {
    api.getOAuthIds("<token>", {}, { project_id: project?.id }).then((res) => {
      let integrations = res.data.filter((integration: any) => {
        return integration.client === "do";
      });

      if (Array.isArray(integrations) && integrations.length) {
        // Sort decendant
        integrations.sort((a, b) => b.id - a.id);
        let lastUsed = integrations.find((i) => {
          i.id === snap.StateHandler?.provision_resources?.credentials?.id;
        });
        if (!lastUsed) {
          lastUsed = integrations[0];
        }
        setConnectedAccount(lastUsed);
      }
      setIsLoading(false);
    });
  }, []);

  const submit = (integrationId: number) => {
    nextFormStep({
      credentials: {
        id: integrationId,
      },
    });
  };

  const url = `${window.location.protocol}//${window.location.host}${window.location.pathname}`;

  const encoded_redirect_uri = encodeURIComponent(url);

  if (isLoading) {
    return <Loading />;
  }

  let content = "Project name: n/a";

  if (connectedAccount?.target_email) {
    content = `${connectedAccount?.target_email}`;
  }

  if (connectedAccount?.target_id) {
    content = `${connectedAccount?.target_id}`;
  }

  return (
    <>
      {connectedAccount !== null && (
        <>
          <Helper>Connected account:</Helper>
          <PreviewRow>
            <Flex>
              <i className="material-icons">account_circle</i>
              {content}
            </Flex>
            <Right>
              Connected at {readableDate(connectedAccount.created_at)}
            </Right>
          </PreviewRow>
        </>
      )}
      {connectedAccount !== null ? (
        <Helper>
          Want to use a different account?{" "}
          <A
            href={`/api/projects/${project?.id}/oauth/digitalocean?redirect_uri=${encoded_redirect_uri}`}
          >
            Sign in to DigitalOcean
          </A>
          .
        </Helper>
      ) : (
        <ConnectDigitalOceanButton
          href={`/api/projects/${project?.id}/oauth/digitalocean?redirect_uri=${encoded_redirect_uri}`}
        >
          Sign In to DigitalOcean
        </ConnectDigitalOceanButton>
      )}

      <Br height="5px" />
      {connectedAccount !== null && (
        <SaveButton
          text="Continue"
          disabled={false}
          onClick={() => submit(connectedAccount.id)}
          makeFlush={true}
          clearPosition={true}
          status={""}
          statusPosition={"right"}
        />
      )}
    </>
  );
};

export const SettingsForm: React.FC<{
  nextFormStep: (data: Partial<DOProvisionerConfig>) => void;
  project: any;
}> = ({ nextFormStep, project }) => {
  const snap = useSnapshot(OFState);
  const [buttonStatus, setButtonStatus] = useState("");
  const [tier, setTier] = useState("basic");
  const [region, setRegion] = useState("nyc1");
  const [clusterName, setClusterName] = useState(`${project.name}-cluster`);
  const [currDOKSInfra, setCurrDOKSInfra] = useState<Infrastructure>();
  const [currDOCRInfra, setCurrDOCRInfra] = useState<Infrastructure>();

  useEffect(() => {
    if (!project) {
      return;
    }

    api
      .getInfra<Infrastructure[]>("<token>", {}, { project_id: project.id })
      .then(({ data }) => {
        let sortFunc = (a: Infrastructure, b: Infrastructure) => {
          return b.id < a.id ? -1 : b.id > a.id ? 1 : 0;
        };

        const matchedDOKSInfras = data
          .filter((infra) => infra.kind == "doks")
          .sort(sortFunc);
        const matchedDOCRInfras = data
          .filter((infra) => infra.kind == "docr")
          .sort(sortFunc);

        if (matchedDOKSInfras.length > 0) {
          // get the infra with latest operation details from the API
          api
            .getInfraByID(
              "<token>",
              {},
              { project_id: project.id, infra_id: matchedDOKSInfras[0].id }
            )
            .then(({ data }) => {
              setCurrDOKSInfra(data);
            })
            .catch((err) => {
              console.error(err);
            });
        }

        if (matchedDOCRInfras.length > 0) {
          api
            .getInfraByID(
              "<token>",
              {},
              { project_id: project.id, infra_id: matchedDOCRInfras[0].id }
            )
            .then(({ data }) => {
              setCurrDOCRInfra(data);
            })
            .catch((err) => {
              console.error(err);
            });
        }
      })
      .catch((err) => {});
  }, [project]);

  const validate = () => {
    if (!clusterName) {
      return {
        hasError: true,
        error: "Cluster name cannot be empty",
      };
    }
    if (clusterName.length > 25) {
      return {
        hasError: true,
        error: "Cluster name cannot be longer than 25 characters",
      };
    }
    return {
      hasError: false,
      error: "",
    };
  };

  const catchError = (error: any) => {
    console.error(error);
  };

  const hasRegistryProvisioned = (
    infras: { kind: string; status: string }[]
  ) => {
    return !!infras.find(
      (i) => ["docr", "docr", "ecr"].includes(i.kind) && i.status === "created"
    );
  };

  const hasClusterProvisioned = (
    infras: { kind: string; status: string }[]
  ) => {
    return !!infras.find(
      (i) => ["doks", "gks", "eks"].includes(i.kind) && i.status === "created"
    );
  };

  const provisionDOCR = async (integrationId: number, tier: string) => {
    console.log("Provisioning DOCR...");

    // See if there's an infra for DOKS that is in an errored state and the last operation
    // was an attempt at creation. If so, re-use that infra.
    if (
      currDOCRInfra?.latest_operation?.type == "create" ||
      currDOCRInfra?.latest_operation?.type == "retry_create"
    ) {
      try {
        const res = await api.retryCreateInfra(
          "<token>",
          {
            do_integration_id: integrationId,
            values: {
              docr_name: project.name,
              docr_subscription_tier: tier,
            },
          },
          { project_id: project.id, infra_id: currDOCRInfra.id }
        );
        return res?.data;
      } catch (error) {
        return catchError(error);
      }
    } else {
      try {
        return await api
          .provisionInfra(
            "<token>",
            {
              kind: "docr",
              do_integration_id: integrationId,
              values: {
                docr_name: project.name,
                docr_subscription_tier: tier,
              },
            },
            {
              project_id: project.id,
            }
          )
          .then((res) => res?.data);
      } catch (error) {
        catchError(error);
      }
    }
  };

  const provisionDOKS = async (
    integrationId: number,
    region: string,
    clusterName: string
  ) => {
    console.log("Provisioning DOKS...");

    // See if there's an infra for DOKS that is in an errored state and the last operation
    // was an attempt at creation. If so, re-use that infra.
    if (
      currDOKSInfra?.latest_operation?.type == "create" ||
      currDOKSInfra?.latest_operation?.type == "retry_create"
    ) {
      try {
        const res = await api.retryCreateInfra(
          "<token>",
          {
            do_integration_id: integrationId,
            values: {
              cluster_name: clusterName,
              do_region: region,
              issuer_email: snap.StateHandler.user_email,
            },
          },
          { project_id: project.id, infra_id: currDOKSInfra.id }
        );
        return res?.data;
      } catch (error) {
        return catchError(error);
      }
    } else {
      try {
        return await api
          .provisionInfra(
            "<token>",
            {
              kind: "doks",
              do_integration_id: integrationId,
              values: {
                cluster_name: clusterName,
                do_region: region,
                issuer_email: snap.StateHandler.user_email,
              },
            },
            {
              project_id: project.id,
            }
          )
          .then((res) => res?.data);
      } catch (error) {
        catchError(error);
      }
    }
  };

  const submit = async () => {
    const validation = validate();

    if (validation.hasError) {
      setButtonStatus(validation.error);
      return;
    }

    let infras = [];
    try {
      infras = await api
        .getInfra("<token>", {}, { project_id: project?.id })
        .then((res) => res?.data);
    } catch (error) {
      setButtonStatus("Something went wrong, try again later");
      return;
    }

    const integrationId = snap.StateHandler.provision_resources.credentials.id;
    let registryProvisionResponse = null;
    let clusterProvisionResponse = null;

    if (snap.StateHandler.connected_registry.skip) {
      if (!hasRegistryProvisioned(infras)) {
        registryProvisionResponse = await provisionDOCR(integrationId, tier);
      }
    }

    if (!hasClusterProvisioned(infras)) {
      clusterProvisionResponse = await provisionDOKS(
        integrationId,
        region,
        clusterName
      );
    }

    nextFormStep({
      settings: {
        region,
        tier,
        cluster_name: clusterName,
        registry_infra_id: registryProvisionResponse?.id,
        cluster_infra_id: clusterProvisionResponse?.id,
      },
    });
  };

  return (
    <>
      <SelectRow
        options={tierOptions}
        width="100%"
        value={tier}
        setActiveValue={(x: string) => {
          setTier(x);
        }}
        label="💰 Subscription Tier"
      />
      <SelectRow
        options={regionOptions}
        width="100%"
        dropdownMaxHeight="240px"
        value={region}
        setActiveValue={(x: string) => {
          setRegion(x);
        }}
        label="📍 DigitalOcean Region"
      />
      <InputRow
        type="text"
        value={clusterName}
        setValue={(x: string) => {
          setClusterName(x);
        }}
        label="Cluster Name"
        placeholder="ex: porter-cluster"
        width="100%"
        isRequired={true}
      />
      <Br />
      <SaveButton
        text="Provision resources"
        disabled={false}
        onClick={submit}
        makeFlush={true}
        clearPosition={true}
        status={buttonStatus}
        statusPosition={"right"}
      />
    </>
  );
};

const Right = styled.div`
  text-align: right;
  margin-left: 10px;
`;

const A = styled.a`
  cursor: pointer;
`;

const Flex = styled.div`
  display: flex;
  color: #ffffff;
  align-items: center;
  > i {
    color: #aaaabb;
    font-size: 20px;
    margin-right: 10px;
  }
`;

const PreviewRow = styled.div`
  display: flex;
  align-items: center;
  padding: 12px 15px;
  color: #ffffff55;
  background: #ffffff11;
  border: 1px solid #aaaabb;
  justify-content: space-between;
  font-size: 13px;
  border-radius: 5px;
`;

const Br = styled.div<{ height?: string }>`
  width: 100%;
  height: ${(props) => props.height || "15px"};
`;

const CodeBlock = styled.span`
  display: inline-block;
  background-color: #1b1d26;
  color: white;
  border-radius: 5px;
  font-family: monospace;
  padding: 2px 3px;
  margin-top: -2px;
  user-select: text;
`;

const ConnectDigitalOceanButton = styled.a`
  width: 200px;
  justify-content: center;
  border-radius: 5px;
  display: flex;
  flex-direction: row;
  align-items: center;
  font-size: 13px;
  margin-top: 22px;
  cursor: pointer;
  font-family: "Work Sans", sans-serif;
  color: white;
  font-weight: 500;
  padding: 10px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  box-shadow: 0 5px 8px 0px #00000010;
  cursor: ${(props: { disabled?: boolean }) =>
    props.disabled ? "not-allowed" : "pointer"};

  background: ${(props: { disabled?: boolean }) =>
    props.disabled ? "#aaaabbee" : "#616FEEcc"};
  :hover {
    background: ${(props: { disabled?: boolean }) =>
      props.disabled ? "" : "#505edddd"};
  }

  > i {
    color: white;
    width: 18px;
    height: 18px;
    font-weight: 600;
    font-size: 12px;
    border-radius: 20px;
    display: flex;
    align-items: center;
    margin-right: 5px;
    justify-content: center;
  }
`;
