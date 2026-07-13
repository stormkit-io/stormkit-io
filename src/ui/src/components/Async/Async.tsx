import React, { Suspense } from "react";
import Spinner from "~/components/Spinner";
import "./Async.css";

interface Props {
  children: React.ReactNode;
}

const Async = (
  dynamicImport: () => Promise<{ default: React.FC }>,
  layoutImport?: () => Promise<{ default: React.FC<Props> }>
): React.ReactNode => {
  const Component = React.lazy(dynamicImport);
  let LayoutComponent: React.LazyExoticComponent<React.FC<Props>> | null = null;

  if (layoutImport) {
    LayoutComponent = React.lazy(layoutImport);
  }

  return (
    <Suspense fallback={<Spinner pageCenter />}>
      {LayoutComponent ? (
        <LayoutComponent>
          <Component />
        </LayoutComponent>
      ) : (
        <Component />
      )}
    </Suspense>
  );
};

export default Async;
