/* eslint-disable */
// This code is autogenerated using @harnessio/oats-cli.
// Please do not modify this code directly.
import type { Incident } from '../schemas/Incident';
import type { Pagination } from '../schemas/Pagination';

export interface ListResponse {
  content?: Incident[];
  correlationID?: string;
  pagination?: Pagination;
}
