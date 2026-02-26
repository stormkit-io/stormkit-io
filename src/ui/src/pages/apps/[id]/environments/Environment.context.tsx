import { useState, createContext, useContext, ReactNode } from "react";
import { useParams } from "react-router-dom";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import EmptyList from "~/components/EmptyPage";
import Error404 from "~/components/Errors/Error404";
import { AppContext } from "~/pages/apps/[id]/App.context";
import EnvironmentFormModal from "./_components/EnvironmentFormModal";

export interface EnvironmentContextProps {
  environment: Environment;
}

export const EnvironmentContext = createContext<EnvironmentContextProps>({
  environment: {} as Environment,
});

interface Props {
  children: ReactNode;
}

export default function Provider({ children }: Props) {
  const { envId } = useParams();
  const { app, environments } = useContext(AppContext);
  const [isModalOpen, toggleModal] = useState(false);

  if (environments?.length === 0) {
    return (
      <EmptyList>
        <Typography variant="h2" gutterBottom>
          No environments yet
        </Typography>
        <Typography sx={{ maxWidth: 500, mx: "auto", mb: 2 }}>
          Environments are used to deploy applications based on different
          configurations. For instance, in a monorepo application, frontend and
          backend configurations, or in an SPA, production and development
          configurations, etc.
        </Typography>
        <Button
          variant="contained"
          color="secondary"
          onClick={() => toggleModal(true)}
        >
          Create Environment
        </Button>
        <EnvironmentFormModal
          app={app}
          isOpen={isModalOpen}
          onClose={() => {
            toggleModal(false);
          }}
        />
      </EmptyList>
    );
  }

  const environment = environments?.filter(e => e.id === envId)?.[0];

  if (!environment) {
    return <Error404 withLogo={false}>This environment is not found.</Error404>;
  }

  return (
    <EnvironmentContext.Provider value={{ environment }}>
      {children}
    </EnvironmentContext.Provider>
  );
}
