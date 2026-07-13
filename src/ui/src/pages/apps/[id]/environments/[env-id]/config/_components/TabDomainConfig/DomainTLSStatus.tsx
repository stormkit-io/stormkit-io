// import React, { useEffect, useState } from "react";
// import Link from "~/components/Link";
// import Spinner from "~/components/Spinner";
// import InfoBox from "~/components/InfoBox";

// interface Props {
//   domain: Domain;
// }

// const DomainUsageStatus: React.FC<Props> = ({ domain }) => {
//   const [retry, setRetry] = useState(0);
//   const isVerified = domain.dns?.verified;
//   const cert = domain?.tls;

//   useEffect(() => {
//     let unmounted = false;

//     // Allow a bit time to issue a certificate.
//     setTimeout(() => {
//       if (cert === null && isVerified && !unmounted) {
//         setRetry(retry + 1);
//       }
//     }, 5000);

//     return () => {
//       unmounted = true;
//     };
//   }, [cert, isVerified]);

//   if (!isVerified) {
//     return null;
//   }

//   return (
//     <div>
//       {cert !== null ? (
//         <div className="text-gray-80 w-full">
//           <div className="flex items-center mb-4 border-b border-blue-20">
//             <div className="bg-black p-4 md:min-w-56">TLS Issuer</div>
//             <div className="p-4">
//               {cert?.issuer?.organization?.[0]} {cert?.issuer?.name}
//             </div>
//           </div>
//           <div className="flex items-center mb-4 border-b border-blue-20">
//             <div className="bg-black p-4 md:min-w-56">Issued at</div>
//             <div className="p-4">
//               {new Date(cert.startDate * 1000).toLocaleDateString("en", {
//                 year: "numeric",
//                 month: "long",
//                 day: "2-digit",
//               })}
//             </div>
//           </div>
//           <div className="flex items-center mb-4 border-b border-blue-20">
//             <div className="bg-black p-4 md:min-w-56">Valid until</div>
//             <div className="p-4">
//               {new Date(cert.endDate * 1000).toLocaleDateString("en", {
//                 year: "numeric",
//                 month: "long",
//                 day: "2-digit",
//               })}
//             </div>
//           </div>
//           <div className="flex items-center mb-4 border-b border-blue-20">
//             <div className="bg-black p-4 md:min-w-56">Serial no</div>
//             <div className="p-4">{cert.serialNo}</div>
//           </div>
//           <div className="flex items-center border-b border-blue-20">
//             <div className="bg-black p-4 md:min-w-56">Signature algorithm</div>
//             <div className="p-4">{cert.signatureAlgorithm}</div>
//           </div>
//         </div>
//       ) : (
//         <>
//           <InfoBox className="mb-4">
//             For the TLS certificate to be issued, you have to publish a
//             deployment in this environment.
//           </InfoBox>
//           {retry > 2 && (
//             <InfoBox className="mb-4">
//               Issuing certificate is taking longer than expected. Have you set
//               the CNAME properly and published a deployment to this environment?
//             </InfoBox>
//           )}
//           <div className="text-blue-40 flex items-center">
//             <span className="fas fa-running mr-1" />
//             Certificate is being issued, if it takes a while visit the domain:{" "}
//             <Link
//               to={`https://${domain.domainName}`}
//               className="ml-1"
//               secondary
//             >
//               {domain.domainName}
//             </Link>{" "}
//             <Spinner className="flex ml-2" primary width={6} height={6} />
//           </div>
//         </>
//       )}
//     </div>
//   );
// };

// export default DomainUsageStatus;
