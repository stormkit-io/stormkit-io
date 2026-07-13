import React, { useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";

const Id: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();

  useEffect(() => {
    navigate(`/apps/${id}/environments`);
  }, []);

  return null;
};

export default Id;
