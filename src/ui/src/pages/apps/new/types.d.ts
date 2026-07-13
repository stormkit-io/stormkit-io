export interface Repo {
  name: string;
  fullName: string;
}

export interface Account {
  id: string;
  login: string;
  avatar: string;
}

export interface FetchReposValue {
  loading: boolean;
  repos: Repo[];
  accounts: Account[];
  selectedAccount?: string;
  setSelectedAccount: (id: string) => void;
  error?: string;
  hasNextPage: boolean;
}
