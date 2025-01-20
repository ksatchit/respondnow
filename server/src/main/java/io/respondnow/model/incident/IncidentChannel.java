package io.respondnow.model.incident;

import lombok.Data;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Data
@NoArgsConstructor
@Getter
@Setter
public class IncidentChannel {
  private IncidentChannelType type;
  private Slack slack;
}
