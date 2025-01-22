/* eslint-disable */
// This code is autogenerated using @harnessio/oats-cli.
// Please do not modify this code directly.

/**
 * User data object
 */

export interface LoginResponseData {
  /**
   * Indicates if the user needs to change their password
   * @example true
   */
  changeUserPassword?: boolean;
  /**
   * Timestamp of the user's last login
   * @format int64
   * @example 1630421333000
   */
  lastLoginAt?: number;
  /**
   * JWT token for the user
   */
  token?: string;
}
