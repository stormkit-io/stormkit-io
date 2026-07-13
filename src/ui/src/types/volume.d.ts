declare type VolumeMountType =
  | "filesys"
  | "aws:s3"
  | "alibaba:oss"
  | "hcloud:oss";

declare interface VolumeConfig {
  mountType?: VolumeMountType;
  accessKey?: string;
  secretKey?: string;
  bucketName?: string;
  region?: string;
  rootPath?: string;
}

declare interface VolumeFile {
  id: string;
  name: string;
  size: number;
  isPublic: boolean;
  createdAt: number;
  updatedAt?: number;
  publicLink?: string;
}
