import { useContext, useEffect } from "react";
import { AuthContext } from "~/pages/auth/Auth.context";

const Logout: React.FC = (): null => {
  const { logout } = useContext(AuthContext);

  useEffect(() => {
    logout?.();
  }, [logout]);

  return null;
};

export default Logout;
