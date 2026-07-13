import React from "react";

interface Props {
  children: React.ReactNode;
}

const Title: React.FC<Props> & {
  Main: React.FC<Props>;
  Sub: React.FC<Props>;
} = ({ children }): React.ReactElement => {
  return <div className="font-bold">{children}</div>;
};

Title.Main = ({ children }): React.ReactElement => {
  return <h1 className="text-lg text-pink-50 mb-2">{children}</h1>;
};

Title.Sub = ({ children }) => {
  return <h2 className="text-2xl text-white mb-2 lg:mb-6">{children}</h2>;
};

export default Title;
