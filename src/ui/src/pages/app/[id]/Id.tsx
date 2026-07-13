import React, { useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";

const Id = (): React.ReactElement => {
  const { id } = useParams();
  const navigate = useNavigate();

  useEffect(() => {
    navigate(`/apps/${id}`);
  }, []);

  return <></>;
};

export default Id;
